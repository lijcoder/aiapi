package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

const (
	PricingConfigVersion = 2
	defaultPricingZone   = "Asia/Shanghai"
	maxPricingDepth      = 32
	maxPricingNodes      = 256
)

// PricingPrice 是每百万 Token 的三类单价。
type PricingPrice struct {
	InputCacheHit  float64 `json:"input_cache_hit"`
	InputCacheMiss float64 `json:"input_cache_miss"`
	Output         float64 `json:"output"`
}

// PricingConfig 是一个模型完整的计费配置。
type PricingConfig struct {
	Version      int           `json:"version"`
	Timezone     string        `json:"timezone"`
	DefaultPrice PricingPrice  `json:"default_price"`
	Rules        []PricingRule `json:"rules"`
}

// PricingRule 是一条按优先级匹配的计费规则。ID 仅用于兼容旧配置，v2 不再输出它。
type PricingRule struct {
	ID       string           `json:"-"` // 仅用于读取旧配置，不再写入 v2 配置。
	Name     string           `json:"name"`
	Enabled  bool             `json:"enabled"`
	Priority int              `json:"priority"`
	When     PricingCondition `json:"when"`
	Price    PricingPrice     `json:"price"`
}

// PricingCondition 是可递归组合的条件树节点。
// group 节点使用 op=and/or 和 children；叶子节点使用 type 及对应字段。
type PricingCondition struct {
	Op        string             `json:"op,omitempty"`
	Children  []PricingCondition `json:"children,omitempty"`
	Type      string             `json:"type,omitempty"`
	Values    []int              `json:"values,omitempty"`
	Start     string             `json:"start,omitempty"`
	End       string             `json:"end,omitempty"`
	MonthDays []string           `json:"month_days,omitempty"`
	Operator  string             `json:"operator,omitempty"`
	Value     int                `json:"value,omitempty"`
}

// PricingWhen 保留为别名，便于调用方平滑迁移到条件树。
type PricingWhen = PricingCondition

// 旧版配置兼容类型。
type PricingTimeWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type PricingTimeCondition struct {
	Weekdays []int               `json:"weekdays"`
	Windows  []PricingTimeWindow `json:"windows"`
}

type PricingTokenCondition struct {
	GT  *int `json:"gt,omitempty"`
	GTE *int `json:"gte,omitempty"`
	LT  *int `json:"lt,omitempty"`
	LTE *int `json:"lte,omitempty"`
}

// PricingRuleSnapshot 是用量记录中保留的已命中规则信息。
type PricingRuleSnapshot struct {
	ID       string           `json:"id,omitempty"` // 仅旧版规则快照可能有值；v2 以 name 为准。
	Name     string           `json:"name"`
	Priority int              `json:"priority"`
	When     PricingCondition `json:"when"`
}

// PricingSnapshot 固化一次请求实际使用的价格与规则，供之后审计。
type PricingSnapshot struct {
	Version          int                  `json:"version"`
	Source           string               `json:"source"`
	ModelID          int64                `json:"model_id"`
	ConfigVersion    int                  `json:"config_version"`
	Timezone         string               `json:"timezone"`
	RequestStartedAt string               `json:"request_started_at"`
	MatchedRuleName  string               `json:"matched_rule_name"`
	Rule             *PricingRuleSnapshot `json:"rule,omitempty"`
	Price            PricingPrice         `json:"price"`
	PricingConfig    PricingConfig        `json:"pricing_config"`
}

type PricingQuote struct {
	Price    PricingPrice
	Snapshot PricingSnapshot
}

// UnmarshalJSON 严格解析条件节点，避免管理端误传未知字段。
func (c *PricingCondition) UnmarshalJSON(data []byte) error {
	type plain PricingCondition
	var value plain
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil {
		return err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return err
	}
	*c = PricingCondition(value)
	return nil
}

// UnmarshalJSON 解析 v2 规则，并把 v1 的平铺 when 转成条件树。
func (r *PricingRule) UnmarshalJSON(data []byte) error {
	var wire struct {
		ID       string          `json:"id"`
		Name     string          `json:"name"`
		Enabled  bool            `json:"enabled"`
		Priority int             `json:"priority"`
		When     json.RawMessage `json:"when"`
		Price    PricingPrice    `json:"price"`
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return err
	}
	when, err := parsePricingWhen(wire.When)
	if err != nil {
		return err
	}
	*r = PricingRule{ID: strings.TrimSpace(wire.ID), Name: wire.Name, Enabled: wire.Enabled, Priority: wire.Priority, When: when, Price: wire.Price}
	return nil
}

func parsePricingWhen(raw json.RawMessage) (PricingCondition, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return PricingCondition{}, nil
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return PricingCondition{}, err
	}
	if _, oldTime := probe["time"]; oldTime {
		var legacy struct {
			Time        *PricingTimeCondition  `json:"time"`
			TotalTokens *PricingTokenCondition `json:"total_tokens"`
		}
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&legacy); err != nil {
			return PricingCondition{}, err
		}
		if err := validateLegacyWhen(legacy.Time, legacy.TotalTokens); err != nil {
			return PricingCondition{}, err
		}
		return legacyWhenToCondition(legacy.Time, legacy.TotalTokens), nil
	}
	if _, oldTokens := probe["total_tokens"]; oldTokens {
		var legacy struct {
			Time        *PricingTimeCondition  `json:"time"`
			TotalTokens *PricingTokenCondition `json:"total_tokens"`
		}
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&legacy); err != nil {
			return PricingCondition{}, err
		}
		if err := validateLegacyWhen(legacy.Time, legacy.TotalTokens); err != nil {
			return PricingCondition{}, err
		}
		return legacyWhenToCondition(legacy.Time, legacy.TotalTokens), nil
	}
	var condition PricingCondition
	if err := json.Unmarshal(raw, &condition); err != nil {
		return PricingCondition{}, err
	}
	return condition, nil
}

func validateLegacyWhen(tm *PricingTimeCondition, tokens *PricingTokenCondition) error {
	if tm != nil {
		if len(tm.Weekdays) == 0 || len(tm.Windows) == 0 {
			return fmt.Errorf("旧版时间条件必须同时设置 weekdays 和 windows")
		}
		for _, window := range tm.Windows {
			start, err := parsePricingClock(window.Start)
			if err != nil {
				return err
			}
			end, err := parsePricingClock(window.End)
			if err != nil || start >= end {
				return fmt.Errorf("旧版时间段无效")
			}
		}
	}
	if tokens != nil {
		for _, bound := range []*int{tokens.GT, tokens.GTE, tokens.LT, tokens.LTE} {
			if bound != nil && *bound < 0 {
				return fmt.Errorf("旧版 total_tokens 边界不能小于 0")
			}
		}
	}
	return nil
}

func legacyWhenToCondition(tm *PricingTimeCondition, tokens *PricingTokenCondition) PricingCondition {
	children := make([]PricingCondition, 0, 2)
	if tm != nil {
		timeChildren := make([]PricingCondition, 0, 2)
		if len(tm.Weekdays) > 0 {
			timeChildren = append(timeChildren, PricingCondition{Type: "weekday", Values: tm.Weekdays})
		}
		windows := make([]PricingCondition, 0, len(tm.Windows))
		for _, window := range tm.Windows {
			windows = append(windows, PricingCondition{Type: "time_range", Start: window.Start, End: window.End})
		}
		if len(windows) == 1 {
			timeChildren = append(timeChildren, windows[0])
		} else if len(windows) > 1 {
			timeChildren = append(timeChildren, PricingCondition{Op: "or", Children: windows})
		}
		children = append(children, PricingCondition{Op: "and", Children: timeChildren})
	}
	if tokens != nil {
		for _, item := range []struct {
			op    string
			value *int
		}{{"gt", tokens.GT}, {"gte", tokens.GTE}, {"lt", tokens.LT}, {"lte", tokens.LTE}} {
			if item.value != nil {
				children = append(children, PricingCondition{Type: "total_tokens", Operator: item.op, Value: *item.value})
			}
		}
	}
	if len(children) == 1 {
		return children[0]
	}
	return PricingCondition{Op: "and", Children: children}
}

func ParsePricingConfig(raw string) (*PricingConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("计费配置不能为空")
	}
	var cfg PricingConfig
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("计费配置格式错误: %w", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return nil, err
	}
	if cfg.Version == 1 {
		for i := range cfg.Rules {
			if strings.TrimSpace(cfg.Rules[i].Name) == "" {
				cfg.Rules[i].Name = strings.TrimSpace(cfg.Rules[i].ID)
			}
		}
		cfg.Version = PricingConfigVersion
	}
	if err := ValidatePricingConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func NormalizePricingConfig(raw string) (string, error) {
	cfg, err := ParsePricingConfig(raw)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("计费配置只能包含一个 JSON 对象")
		}
		return fmt.Errorf("计费配置格式错误: %w", err)
	}
	return nil
}

func ValidatePricingConfig(cfg *PricingConfig) error {
	if cfg == nil {
		return fmt.Errorf("计费配置不能为空")
	}
	if cfg.Version != PricingConfigVersion {
		return fmt.Errorf("计费配置版本必须为 %d", PricingConfigVersion)
	}
	if cfg.Timezone == "" {
		cfg.Timezone = defaultPricingZone
	}
	if cfg.Rules == nil {
		cfg.Rules = []PricingRule{}
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return fmt.Errorf("计费时区无效: %s", cfg.Timezone)
	}
	if err := validatePricingPrice(cfg.DefaultPrice, "默认价格"); err != nil {
		return err
	}
	names := make(map[string]struct{}, len(cfg.Rules))
	priorities := make(map[int]struct{}, len(cfg.Rules))
	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		rule.Name = strings.TrimSpace(rule.Name)
		if rule.Name == "" {
			return fmt.Errorf("第 %d 条规则缺少 name", i+1)
		}
		if _, ok := names[rule.Name]; ok {
			return fmt.Errorf("规则 name 重复: %s", rule.Name)
		}
		names[rule.Name] = struct{}{}
		if _, ok := priorities[rule.Priority]; ok {
			return fmt.Errorf("规则 priority 重复: %d", rule.Priority)
		}
		priorities[rule.Priority] = struct{}{}
		nodes := 0
		if err := validatePricingCondition(rule.When, 1, &nodes); err != nil {
			return fmt.Errorf("规则 %s: %w", rule.Name, err)
		}
		if err := validatePricingPrice(rule.Price, "规则价格"); err != nil {
			return fmt.Errorf("规则 %s: %w", rule.Name, err)
		}
	}
	return nil
}

func validatePricingPrice(price PricingPrice, name string) error {
	for label, value := range map[string]float64{"缓存命中输入价": price.InputCacheHit, "缓存未命中输入价": price.InputCacheMiss, "输出价": price.Output} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("%s的%s必须是非负有限数", name, label)
		}
	}
	return nil
}

func validatePricingCondition(condition PricingCondition, depth int, nodes *int) error {
	if depth > maxPricingDepth {
		return fmt.Errorf("条件嵌套层级不能超过 %d", maxPricingDepth)
	}
	*nodes = *nodes + 1
	if *nodes > maxPricingNodes {
		return fmt.Errorf("条件节点数不能超过 %d", maxPricingNodes)
	}
	if condition.Op != "" {
		op := strings.ToLower(condition.Op)
		if op != "and" && op != "or" {
			return fmt.Errorf("条件组 op 必须是 and 或 or")
		}
		if condition.Type != "" {
			return fmt.Errorf("条件组不能设置 type")
		}
		if len(condition.Children) == 0 {
			return fmt.Errorf("条件组 children 不能为空")
		}
		for _, child := range condition.Children {
			if err := validatePricingCondition(child, depth+1, nodes); err != nil {
				return err
			}
		}
		if op == "and" {
			if err := validateTokenAndGroup(condition.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if len(condition.Children) > 0 {
		return fmt.Errorf("非条件组不能设置 children")
	}
	switch condition.Type {
	case "weekday":
		if len(condition.Values) == 0 {
			return fmt.Errorf("weekday values 不能为空")
		}
		seen := map[int]struct{}{}
		for _, day := range condition.Values {
			if day < 1 || day > 7 {
				return fmt.Errorf("weekday 必须在 1 到 7 之间")
			}
			if _, ok := seen[day]; ok {
				return fmt.Errorf("weekday 不能重复")
			}
			seen[day] = struct{}{}
		}
	case "time_range":
		start, err := parsePricingClock(condition.Start)
		if err != nil {
			return fmt.Errorf("时间段开始时间无效: %w", err)
		}
		end, err := parsePricingClock(condition.End)
		if err != nil {
			return fmt.Errorf("时间段结束时间无效: %w", err)
		}
		if start >= end {
			return fmt.Errorf("时间段必须满足 start < end，不支持跨天时间段")
		}
	case "month_day":
		if len(condition.MonthDays) == 0 {
			return fmt.Errorf("month_day month_days 不能为空")
		}
		seen := map[string]struct{}{}
		for _, value := range condition.MonthDays {
			canonical, err := parsePricingMonthDay(value)
			if err != nil {
				return err
			}
			if _, ok := seen[canonical]; ok {
				return fmt.Errorf("month_day 不能重复")
			}
			seen[canonical] = struct{}{}
		}
	case "total_tokens":
		switch condition.Operator {
		case "gt", "gte", "lt", "lte":
		default:
			return fmt.Errorf("total_tokens operator 必须是 gt、gte、lt 或 lte")
		}
		if condition.Value < 0 {
			return fmt.Errorf("total_tokens value 不能小于 0")
		}
	default:
		return fmt.Errorf("未知条件类型: %s", condition.Type)
	}
	return nil
}

// validateTokenAndGroup 检查 AND 组中可静态推导出的 Token 边界，避免空区间。
func validateTokenAndGroup(children []PricingCondition) error {
	var lower, upper int
	var lowerInclusive, upperInclusive, hasLower, hasUpper bool
	var visit func(PricingCondition)
	visit = func(c PricingCondition) {
		if c.Op == "and" {
			for _, child := range c.Children {
				visit(child)
			}
			return
		}
		if c.Type != "total_tokens" {
			return
		}
		switch c.Operator {
		case "gt", "gte":
			inclusive := c.Operator == "gte"
			if !hasLower || c.Value > lower || (c.Value == lower && !inclusive && lowerInclusive) {
				lower, lowerInclusive, hasLower = c.Value, inclusive, true
			}
		case "lt", "lte":
			inclusive := c.Operator == "lte"
			if !hasUpper || c.Value < upper || (c.Value == upper && !inclusive && upperInclusive) {
				upper, upperInclusive, hasUpper = c.Value, inclusive, true
			}
		}
	}
	for _, child := range children {
		visit(child)
	}
	if hasLower && hasUpper && (lower > upper || (lower == upper && !(lowerInclusive && upperInclusive))) {
		return fmt.Errorf("total_tokens 区间为空")
	}
	return nil
}

func parsePricingClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil || parsed.Format("15:04") != value {
		return 0, fmt.Errorf("%q，格式应为 HH:MM", value)
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func parsePricingMonthDay(value string) (string, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("month_day %q 格式应为 MM-DD", value)
	}
	month, monthErr := parseTwoDigitOrNumber(parts[0])
	day, dayErr := parseTwoDigitOrNumber(parts[1])
	if monthErr != nil || dayErr != nil || month < 1 || month > 12 || day < 1 || day > daysInMonth(month) {
		return "", fmt.Errorf("month_day %q 不是有效日期", value)
	}
	return fmt.Sprintf("%02d-%02d", month, day), nil
}

func parseTwoDigitOrNumber(value string) (int, error) {
	if value == "" || len(value) > 2 {
		return 0, fmt.Errorf("invalid number")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid number")
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func daysInMonth(month int) int {
	if month == 2 {
		return 29
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

func QuoteModelPricing(m *model.Model, requestStartedAt time.Time, totalTokens int) (*PricingQuote, error) {
	if m == nil {
		return nil, fmt.Errorf("模型不能为空")
	}
	cfg, err := ParsePricingConfig(m.PricingConfig)
	if err != nil {
		return nil, err
	}
	loc, _ := time.LoadLocation(cfg.Timezone)
	price := cfg.DefaultPrice
	snapshot := PricingSnapshot{Version: PricingConfigVersion, Source: "default", ModelID: m.ID, ConfigVersion: cfg.Version, Timezone: cfg.Timezone, RequestStartedAt: requestStartedAt.In(loc).Format("2006-01-02 15:04:05"), MatchedRuleName: "默认价格", Price: price, PricingConfig: *cfg}
	matched := make([]PricingRule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		if rule.Enabled && rule.When.matches(requestStartedAt, loc, totalTokens) {
			matched = append(matched, rule)
		}
	}
	if len(matched) > 0 {
		sort.SliceStable(matched, func(i, j int) bool { return matched[i].Priority > matched[j].Priority })
		rule := matched[0]
		price = rule.Price
		snapshot.Source = "rule"
		snapshot.MatchedRuleName = rule.Name
		snapshot.Price = price
		snapshot.Rule = &PricingRuleSnapshot{ID: rule.ID, Name: rule.Name, Priority: rule.Priority, When: rule.When}
	}
	return &PricingQuote{Price: price, Snapshot: snapshot}, nil
}

func (condition PricingCondition) matches(requestStartedAt time.Time, loc *time.Location, totalTokens int) bool {
	if condition.Op != "" {
		if strings.EqualFold(condition.Op, "and") {
			for _, child := range condition.Children {
				if !child.matches(requestStartedAt, loc, totalTokens) {
					return false
				}
			}
			return true
		}
		for _, child := range condition.Children {
			if child.matches(requestStartedAt, loc, totalTokens) {
				return true
			}
		}
		return false
	}
	now := requestStartedAt.In(loc)
	switch condition.Type {
	case "weekday":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		for _, day := range condition.Values {
			if day == weekday {
				return true
			}
		}
	case "time_range":
		start, _ := parsePricingClock(condition.Start)
		end, _ := parsePricingClock(condition.End)
		minute := now.Hour()*60 + now.Minute()
		return minute >= start && minute < end
	case "month_day":
		current := fmt.Sprintf("%02d-%02d", int(now.Month()), now.Day())
		for _, value := range condition.MonthDays {
			canonical, _ := parsePricingMonthDay(value)
			if canonical == current {
				return true
			}
		}
	case "total_tokens":
		switch condition.Operator {
		case "gt":
			return totalTokens > condition.Value
		case "gte":
			return totalTokens >= condition.Value
		case "lt":
			return totalTokens < condition.Value
		case "lte":
			return totalTokens <= condition.Value
		}
	}
	return false
}

type UsageBillingInput struct {
	Model            *model.Model
	UserID           int64
	APIKeyID         int64
	Provider         string
	ModelName        string
	InputTokens      int
	OutputTokens     int
	TotalTokens      int
	RequestID        string
	Stream           bool
	CachedTokens     int
	ReasoningTokens  int
	UserUnlimited    bool
	KeyUnlimited     bool
	FirstTokenMs     int64
	LatencyMs        int64
	RequestStartedAt time.Time
}

type BillingService struct{}

func NewBillingService() *BillingService { return &BillingService{} }

func (s *BillingService) RecordUsage(input UsageBillingInput) (*model.UsageRecord, error) {
	if input.InputTokens < 0 || input.OutputTokens < 0 || input.TotalTokens < 0 || input.CachedTokens < 0 || input.CachedTokens > input.InputTokens || input.ReasoningTokens < 0 {
		return nil, fmt.Errorf("用量数据无效")
	}
	quote, err := QuoteModelPricing(input.Model, input.RequestStartedAt, input.TotalTokens)
	if err != nil {
		return nil, err
	}
	snapshot, err := json.Marshal(quote.Snapshot)
	if err != nil {
		return nil, err
	}
	inputMiss := input.InputTokens - input.CachedTokens
	cost := (float64(input.CachedTokens)*quote.Price.InputCacheHit + float64(inputMiss)*quote.Price.InputCacheMiss + float64(input.OutputTokens)*quote.Price.Output) / 1_000_000
	record := &model.UsageRecord{UserID: input.UserID, ApiKeyID: input.APIKeyID, Provider: input.Provider, Model: input.ModelName, InputTokens: input.InputTokens, OutputTokens: input.OutputTokens, TotalTokens: input.TotalTokens, RequestID: input.RequestID, Stream: input.Stream, CachedTokens: input.CachedTokens, ReasoningTokens: input.ReasoningTokens, Cost: cost, Unlimited: input.UserUnlimited, PricingSnapshot: string(snapshot), FirstTokenMs: input.FirstTokenMs, LatencyMs: input.LatencyMs}
	err = store.C().T(func(session *store.Session) error {
		if err := session.Usage().Insert(record); err != nil {
			return err
		}
		if input.UserUnlimited {
			return nil
		}
		if err := session.Charge().DeductUserBudget(input.UserID, cost); err != nil {
			return err
		}
		if !input.KeyUnlimited {
			return session.Charge().DeductKeyBudget(input.APIKeyID, cost)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}
