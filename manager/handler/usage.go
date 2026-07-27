package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
)

// ===== 请求结构体 =====

// UsageStatsSelfReq 统计汇总请求
type UsageStatsSelfReq struct {
	Mode      string `json:"mode"`       // "day" | "month"
	StartDate string `json:"start_date"` // 含，格式 2006-01-02 或 2006-01
	EndDate   string `json:"end_date"`   // 不含，格式同上
	ApiKeyId  int64  `json:"api_key_id"` // 可选，0=不限
	Model     string `json:"model"`      // 可选，空=不限
	Provider  string `json:"provider"`   // 可选，空=不限
	GroupBy   string `json:"group_by"`   // 可选：""按时间 | "model" | "provider" | "api_key"
}

// UsageFilterOption ApiKey 筛选选项
type UsageFilterOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"` // 脱敏
}

// UsageFiltersResp 筛选选项响应
type UsageFiltersResp struct {
	ApiKeys   []UsageFilterOption `json:"api_keys"`
	Models    []string            `json:"models"`
	Providers []string            `json:"providers"`
}

// UsageSummary 顶部汇总指标
type UsageSummary struct {
	RequestCount    int64   `json:"request_count"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	CachedTokens    int64   `json:"cached_tokens"`
	CacheMissTokens int64   `json:"cache_miss_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	TotalCost       float64 `json:"total_cost"`
	AvgCost         float64 `json:"avg_cost"`
}

// UsageStatsSelfResp 统计汇总响应
type UsageStatsSelfResp struct {
	Summary UsageSummary         `json:"summary"`
	Rows    []store.UsageStatRow `json:"rows"`
}

// ===== 辅助函数 =====

// resolveApiKey 根据 api_key_id 查出完整 key；id<=0 返回空串（不筛选）
func resolveApiKey(userID int64, apiKeyID int64) (string, error) {
	if apiKeyID <= 0 {
		return "", nil
	}
	k, err := store.C().ApiKey().GetByID(apiKeyID)
	if err != nil {
		return "", err
	}
	if k == nil || k.UserID != userID {
		return "", nil // 不属于当前用户，忽略
	}
	return k.Key, nil
}

// parseDateRangeDay 按天模式解析日期；返回 (start含, end含, error)
func parseDateRangeDay(startStr, endStr string) (string, string, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return "", "", err
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return "", "", err
	}
	if end.Before(start) {
		return "", "", errInvalidDateRange
	}
	if end.Sub(start) > 31*24*time.Hour {
		return "", "", errDateRangeTooLong
	}
	return start.Format("2006-01-02") + " 00:00:00", end.Format("2006-01-02") + " 23:59:59", nil
}

// parseDateRangeMonth 按月模式解析日期；返回 (start含, end含, error)
func parseDateRangeMonth(startStr, endStr string) (string, string, error) {
	start, err := time.Parse("2006-01", startStr)
	if err != nil {
		return "", "", err
	}
	end, err := time.Parse("2006-01", endStr)
	if err != nil {
		return "", "", err
	}
	if end.Before(start) {
		return "", "", errInvalidDateRange
	}
	lastDay := end.AddDate(0, 1, -1)
	return start.Format("2006-01-02") + " 00:00:00", lastDay.Format("2006-01-02") + " 23:59:59", nil
}

var (
	errInvalidDateRange = &base.BizError{Code: base.CodeBadRequest, Msg: "日期范围无效"}
	errDateRangeTooLong = &base.BizError{Code: base.CodeBadRequest, Msg: "按天查询最多只能选择 31 天"}
)

// ===== Handler =====

// UsageStatsSelf 统计汇总（按天/月）
func UsageStatsSelf(ctx context.Context, req *UsageStatsSelfReq) (*UsageStatsSelfResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	mode := req.Mode
	if mode == "" {
		mode = "day"
	}
	if mode != "day" && mode != "month" {
		return nil, base.ErrBadReq("mode 只能是 day 或 month")
	}
	if req.GroupBy != "" && req.GroupBy != "model" && req.GroupBy != "provider" && req.GroupBy != "api_key" {
		return nil, base.ErrBadReq("group_by 只能为空、model、provider 或 api_key")
	}

	var startDate, endDate string
	var err error
	if mode == "day" {
		startDate, endDate, err = parseDateRangeDay(req.StartDate, req.EndDate)
	} else {
		startDate, endDate, err = parseDateRangeMonth(req.StartDate, req.EndDate)
	}
	if err != nil {
		return nil, base.ErrBadReq(err.Error())
	}

	apiKey, err := resolveApiKey(cur.ID, req.ApiKeyId)
	if err != nil {
		slog.Error("resolve api key failed", "err", err, "api_key_id", req.ApiKeyId)
		return nil, base.ErrInternal
	}

	rows, err := service.NewUsageService().StatsByUser(cur.ID, mode, startDate, endDate, apiKey, req.Model, req.Provider, req.GroupBy)
	if err != nil {
		slog.Error("usage stats failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}

	// 按 api_key 分组时：脱敏 + 填充名称 + 标记是否已删除
	if req.GroupBy == "api_key" {
		apiKeys, err := store.C().ApiKey().ListByUser(cur.ID)
		if err != nil {
			slog.Error("list api keys for stats failed", "err", err, "user_id", cur.ID)
			return nil, base.ErrInternal
		}
		keyNameMap := make(map[string]string, len(apiKeys))
		for _, k := range apiKeys {
			keyNameMap[k.Key] = k.Name
		}
		for i := range rows {
			if name, ok := keyNameMap[rows[i].Label]; ok {
				rows[i].SubLabel = name
				rows[i].KeyExists = ptrBool(true)
			} else {
				rows[i].SubLabel = "已删除"
				rows[i].KeyExists = ptrBool(false)
			}
			rows[i].Label = maskApiKey(rows[i].Label)
		}
	}

	// 汇总顶部指标
	var s UsageSummary
	for _, r := range rows {
		s.RequestCount += r.RequestCount
		s.InputTokens += r.InputTokens
		s.OutputTokens += r.OutputTokens
		s.CachedTokens += r.CachedTokens
		s.CacheMissTokens += r.CacheMissTokens
		s.ReasoningTokens += r.ReasoningTokens
		s.TotalTokens += r.TotalTokens
		s.TotalCost += r.Cost
	}
	if s.RequestCount > 0 {
		s.AvgCost = s.TotalCost / float64(s.RequestCount)
	}
	if s.InputTokens > 0 {
		s.CacheHitRate = float64(s.CachedTokens) / float64(s.InputTokens)
	}

	if rows == nil {
		rows = []store.UsageStatRow{}
	}
	return &UsageStatsSelfResp{Summary: s, Rows: rows}, nil
}

// UsageFiltersSelf 返回该用户可用的筛选选项（api_key / model / provider）
func UsageFiltersSelf(ctx context.Context) (*UsageFiltersResp, *base.BizError) {
	cur := base.CurrentUser(ctx)

	apiKeys, err := store.C().ApiKey().ListByUser(cur.ID)
	if err != nil {
		slog.Error("list api keys for filters failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	keyOpts := make([]UsageFilterOption, 0, len(apiKeys))
	for _, k := range apiKeys {
		keyOpts = append(keyOpts, UsageFilterOption{
			ID:   k.ID,
			Name: k.Name,
			Key:  maskApiKey(k.Key),
		})
	}

	models, err := store.C().Usage().DistinctModelsByUser(cur.ID)
	if err != nil {
		slog.Error("distinct models failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}

	providers, err := store.C().Usage().DistinctProvidersByUser(cur.ID)
	if err != nil {
		slog.Error("distinct providers failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}

	return &UsageFiltersResp{
		ApiKeys:   keyOpts,
		Models:    models,
		Providers: providers,
	}, nil
}

func maskApiKey(k string) string {
	if len(k) <= 10 {
		return k
	}
	return k[:6] + "****" + k[len(k)-4:]
}

func ptrBool(v bool) *bool { return &v }

// ===== Admin 版（全局统计） =====

// UsageStatsAdminReq 全局统计请求

type UsageStatsAdminReq struct {
	Mode      string `json:"mode"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	UserID    int64  `json:"user_id"`   // 可选，0=不限
	ApiKeyId  int64  `json:"api_key_id"`
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	GroupBy   string `json:"group_by"` // "" | model | provider | api_key | user
}

type UsageStatsAdminResp struct {
	Summary UsageSummary         `json:"summary"`
	Rows    []store.UsageStatRow `json:"rows"`
}

// UsageFiltersAdminResp 全局筛选选项

type UsageFiltersAdminResp struct {
	Users     []store.UsageUserOption `json:"users"`
	ApiKeys   []UsageFilterOption     `json:"api_keys"`
	Models    []string                `json:"models"`
	Providers []string                `json:"providers"`
}

// UsageStatsAdmin 全局统计汇总
func UsageStatsAdmin(ctx context.Context, req *UsageStatsAdminReq) (*UsageStatsAdminResp, *base.BizError) {
	mode := req.Mode
	if mode == "" {
		mode = "day"
	}
	if mode != "day" && mode != "month" {
		return nil, base.ErrBadReq("mode 只能是 day 或 month")
	}
	if req.GroupBy != "" && req.GroupBy != "model" && req.GroupBy != "provider" && req.GroupBy != "api_key" && req.GroupBy != "user" {
		return nil, base.ErrBadReq("group_by 只能为空、model、provider、api_key 或 user")
	}

	var startDate, endDate string
	var err error
	if mode == "day" {
		startDate, endDate, err = parseDateRangeDay(req.StartDate, req.EndDate)
	} else {
		startDate, endDate, err = parseDateRangeMonth(req.StartDate, req.EndDate)
	}
	if err != nil {
		return nil, base.ErrBadReq(err.Error())
	}

	// api_key_id → 完整 key（admin 版不校验归属）
	apiKey, err := resolveApiKeyAdmin(req.ApiKeyId)
	if err != nil {
		slog.Error("resolve api key failed", "err", err, "api_key_id", req.ApiKeyId)
		return nil, base.ErrInternal
	}

	rows, err := service.NewUsageService().StatsByAdmin(req.UserID, mode, startDate, endDate, apiKey, req.Model, req.Provider, req.GroupBy)
	if err != nil {
		slog.Error("usage stats admin failed", "err", err)
		return nil, base.ErrInternal
	}

	// 按 api_key 分组：填充名称 + 脱敏 + 标记是否已删除
	if req.GroupBy == "api_key" {
		allKeys, err := store.C().ApiKey().ListAll()
		if err != nil {
			slog.Error("list all api keys for stats failed", "err", err)
			return nil, base.ErrInternal
		}
		keyNameMap := make(map[string]string, len(allKeys))
		for _, k := range allKeys {
			keyNameMap[k.Key] = k.Name
		}
		for i := range rows {
			if name, ok := keyNameMap[rows[i].Label]; ok {
				rows[i].SubLabel = name
				rows[i].KeyExists = ptrBool(true)
			} else {
				rows[i].SubLabel = ""
				rows[i].KeyExists = ptrBool(false)
			}
			rows[i].Label = maskApiKey(rows[i].Label)
		}
	}

	// 汇总顶部指标
	var s UsageSummary
	for _, r := range rows {
		s.RequestCount += r.RequestCount
		s.InputTokens += r.InputTokens
		s.OutputTokens += r.OutputTokens
		s.CachedTokens += r.CachedTokens
		s.CacheMissTokens += r.CacheMissTokens
		s.ReasoningTokens += r.ReasoningTokens
		s.TotalTokens += r.TotalTokens
		s.TotalCost += r.Cost
	}
	if s.RequestCount > 0 {
		s.AvgCost = s.TotalCost / float64(s.RequestCount)
	}
	if s.InputTokens > 0 {
		s.CacheHitRate = float64(s.CachedTokens) / float64(s.InputTokens)
	}

	if rows == nil {
		rows = []store.UsageStatRow{}
	}
	return &UsageStatsAdminResp{Summary: s, Rows: rows}, nil
}

// UsageFiltersAdmin 全局筛选选项（包含用户列表）
func UsageFiltersAdmin(ctx context.Context) (*UsageFiltersAdminResp, *base.BizError) {
	users, err := store.C().Usage().DistinctUsersByAdmin()
	if err != nil {
		slog.Error("distinct users failed", "err", err)
		return nil, base.ErrInternal
	}

	// api_key 列表（全平台）
	allKeys, err := store.C().ApiKey().ListAll()
	if err != nil {
		slog.Error("list all api keys failed", "err", err)
		return nil, base.ErrInternal
	}
	keyOpts := make([]UsageFilterOption, 0, len(allKeys))
	for _, k := range allKeys {
		keyOpts = append(keyOpts, UsageFilterOption{
			ID:   k.ID,
			Name: k.Name,
			Key:  maskApiKey(k.Key),
		})
	}

	models, err := store.C().Usage().DistinctModelsByAdmin(0)
	if err != nil {
		slog.Error("distinct models failed", "err", err)
		return nil, base.ErrInternal
	}

	providers, err := store.C().Usage().DistinctProvidersByAdmin(0)
	if err != nil {
		slog.Error("distinct providers failed", "err", err)
		return nil, base.ErrInternal
	}

	return &UsageFiltersAdminResp{
		Users:     users,
		ApiKeys:   keyOpts,
		Models:    models,
		Providers: providers,
	}, nil
}

// resolveApiKeyAdmin admin 版：id<=0 返回空串，否则查 key 不校验归属
func resolveApiKeyAdmin(apiKeyID int64) (string, error) {
	if apiKeyID <= 0 {
		return "", nil
	}
	k, err := store.C().ApiKey().GetByID(apiKeyID)
	if err != nil {
		return "", err
	}
	if k == nil {
		return "", nil
	}
	return k.Key, nil
}
