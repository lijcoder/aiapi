package store

import (
	"fmt"
	"time"

	"github.com/lijcoder/aiapi/store/model"
)

// Usage 返回用量记录相关操作的命名空间。
func (s *Session) Usage() *UsageStore {
	return &UsageStore{s: s}
}

// UsageStore 是用量记录相关操作的命名空间。
type UsageStore struct {
	s *Session
}

// Insert 插入用量记录
func (us *UsageStore) Insert(usage *model.UsageRecord) error {
	res, err := us.s.Query(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens, cost, unlimited)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens, :cost, :unlimited)`,
		usage,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	usage.ID = id
	return nil
}

// ===== 统计查询 =====

// UsageStatRow 统计汇总行，label 为分组字段值（日期/模型/提供商/Key）
type UsageStatRow struct {
	Label           string  `db:"label" json:"label"`
	SubLabel        string  `db:"-" json:"sub_label,omitempty"`
	KeyExists       *bool   `db:"-" json:"key_exists,omitempty"`
	InputTokens     int64   `db:"input_tokens" json:"input_tokens"`
	OutputTokens    int64   `db:"output_tokens" json:"output_tokens"`
	CachedTokens    int64   `db:"cached_tokens" json:"cached_tokens"`
	CacheMissTokens int64   `db:"cache_miss_tokens" json:"cache_miss_tokens"`
	ReasoningTokens int64   `db:"reasoning_tokens" json:"reasoning_tokens"`
	TotalTokens     int64   `db:"total_tokens" json:"total_tokens"`
	CacheHitRate    float64 `db:"cache_hit_rate" json:"cache_hit_rate"`
	Cost            float64 `db:"cost" json:"cost"`
	RequestCount    int64   `db:"request_count" json:"request_count"`
}

// StatsByUser 统计汇总：按时间（天/月）或按维度（模型/提供商/api_key）分组
func (us *UsageStore) StatsByUser(userID int64, mode, startDate, endDate, apiKey, mdl, provider, groupBy string) ([]UsageStatRow, error) {
	var labelExpr string
	switch groupBy {
	case "model":
		labelExpr = "model"
	case "provider":
		labelExpr = "provider"
	case "api_key":
		labelExpr = "api_key"
	default:
		// 按时间分组
		switch mode {
		case "month":
			labelExpr = "STRFTIME('%Y-%m', created_at)"
		default:
			labelExpr = "DATE(created_at)"
		}
	}

	where, params := buildUsageWhere(userID, startDate, endDate, apiKey, mdl, provider)

	// 按时间分组时需要 GROUP BY labelExpr；按维度分组时按维度列 GROUP BY
	groupExpr := labelExpr
	if groupBy == "" {
		groupExpr = labelExpr // 时间表达式本身
	}

	query := fmt.Sprintf(
		`SELECT %s AS label, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cached_tokens) AS cached_tokens, SUM(input_tokens) - SUM(cached_tokens) AS cache_miss_tokens, SUM(reasoning_tokens) AS reasoning_tokens, SUM(total_tokens) AS total_tokens, ROUND(CAST(SUM(cached_tokens) AS REAL) / NULLIF(SUM(input_tokens), 0), 4) AS cache_hit_rate, SUM(cost) AS cost, COUNT(*) AS request_count FROM usage_records %s GROUP BY %s ORDER BY label DESC`,
		labelExpr, where, groupExpr,
	)

	var rows []UsageStatRow
	err := us.s.Query(query, params).Select(&rows)
	return rows, err
}

// buildUsageWhere 构造通用 WHERE 子句与参数
func buildUsageWhere(userID int64, startDate, endDate string, apiKey, mdl, provider string) (string, map[string]any) {
	where := "WHERE user_id = :user_id AND created_at >= :start_date AND created_at <= :end_date"
	params := map[string]any{
		"user_id":    userID,
		"start_date": startDate,
		"end_date":   endDate,
	}
	if apiKey != "" {
		where += " AND api_key = :api_key"
		params["api_key"] = apiKey
	}
	if mdl != "" {
		where += " AND model = :model"
		params["model"] = mdl
	}
	if provider != "" {
		where += " AND provider = :provider"
		params["provider"] = provider
	}
	return where, params
}

// DistinctModelsByUser 查询某用户用过的所有 model
func (us *UsageStore) DistinctModelsByUser(userID int64) ([]string, error) {
	var models []string
	err := us.s.Query(
		`SELECT DISTINCT model FROM usage_records WHERE user_id = :user_id AND model != '' ORDER BY model`,
		map[string]any{"user_id": userID},
	).Select(&models)
	return models, err
}

// DistinctProvidersByUser 查询某用户用过的所有 provider
func (us *UsageStore) DistinctProvidersByUser(userID int64) ([]string, error) {
	var providers []string
	err := us.s.Query(
		`SELECT DISTINCT provider FROM usage_records WHERE user_id = :user_id AND provider != '' ORDER BY provider`,
		map[string]any{"user_id": userID},
	).Select(&providers)
	return providers, err
}

// ===== Admin 版（不限定 user_id） =====

// StatsByAdmin 全局统计汇总，可按 user_id 筛选，支持 group_by=user
func (us *UsageStore) StatsByAdmin(userID int64, mode, startDate, endDate, apiKey, mdl, provider, groupBy string) ([]UsageStatRow, error) {
	var labelExpr, groupExpr, join string
	switch groupBy {
	case "model":
		labelExpr = "u.model"
		groupExpr = "u.model"
	case "provider":
		labelExpr = "u.provider"
		groupExpr = "u.provider"
	case "api_key":
		labelExpr = "u.api_key"
		groupExpr = "u.api_key"
	case "user":
		labelExpr = "users.name"
		groupExpr = "u.user_id"
		join = "INNER JOIN users ON users.id = u.user_id"
	default:
		switch mode {
		case "month":
			labelExpr = "STRFTIME('%Y-%m', u.created_at)"
		default:
			labelExpr = "DATE(u.created_at)"
		}
		groupExpr = labelExpr
	}

	where, params := buildUsageAdminWhere(userID, startDate, endDate, apiKey, mdl, provider)

	query := fmt.Sprintf(
		`SELECT %s AS label, SUM(u.input_tokens) AS input_tokens, SUM(u.output_tokens) AS output_tokens, SUM(u.cached_tokens) AS cached_tokens, SUM(u.input_tokens) - SUM(u.cached_tokens) AS cache_miss_tokens, SUM(u.reasoning_tokens) AS reasoning_tokens, SUM(u.total_tokens) AS total_tokens, ROUND(CAST(SUM(u.cached_tokens) AS REAL) / NULLIF(SUM(u.input_tokens), 0), 4) AS cache_hit_rate, SUM(u.cost) AS cost, COUNT(*) AS request_count FROM usage_records u %s %s GROUP BY %s ORDER BY label DESC`,
		labelExpr, join, where, groupExpr,
	)

	var rows []UsageStatRow
	err := us.s.Query(query, params).Select(&rows)
	return rows, err
}

// buildUsageAdminWhere 构造 admin 版 WHERE 子句，user_id<=0 时不筛选
func buildUsageAdminWhere(userID int64, startDate, endDate string, apiKey, mdl, provider string) (string, map[string]any) {
	where := "WHERE u.created_at >= :start_date AND u.created_at <= :end_date"
	params := map[string]any{
		"start_date": startDate,
		"end_date": endDate,
	}
	if userID > 0 {
		where += " AND u.user_id = :user_id"
		params["user_id"] = userID
	}
	if apiKey != "" {
		where += " AND u.api_key = :api_key"
		params["api_key"] = apiKey
	}
	if mdl != "" {
		where += " AND u.model = :model"
		params["model"] = mdl
	}
	if provider != "" {
		where += " AND u.provider = :provider"
		params["provider"] = provider
	}
	return where, params
}

// DistinctModelsByAdmin 全平台用过的所有 model
func (us *UsageStore) DistinctModelsByAdmin(userID int64) ([]string, error) {
	q := `SELECT DISTINCT model FROM usage_records WHERE model != ''`
	params := map[string]any{}
	if userID > 0 {
		q += ` AND user_id = :user_id`
		params["user_id"] = userID
	}
	q += ` ORDER BY model`
	var models []string
	err := us.s.Query(q, params).Select(&models)
	return models, err
}

// DistinctProvidersByAdmin 全平台用过的所有 provider
func (us *UsageStore) DistinctProvidersByAdmin(userID int64) ([]string, error) {
	q := `SELECT DISTINCT provider FROM usage_records WHERE provider != ''`
	params := map[string]any{}
	if userID > 0 {
		q += ` AND user_id = :user_id`
		params["user_id"] = userID
	}
	q += ` ORDER BY provider`
	var providers []string
	err := us.s.Query(q, params).Select(&providers)
	return providers, err
}

// UsageUserOption 用户筛选选项

type UsageUserOption struct {
	ID      int64  `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	Account string `db:"account" json:"account"`
}

// DistinctUsersByAdmin 全平台有用量记录的用户列表
func (us *UsageStore) DistinctUsersByAdmin() ([]UsageUserOption, error) {
	var users []UsageUserOption
	err := us.s.Query(
		`SELECT DISTINCT u.id, u.name, u.account FROM users u LEFT JOIN usage_records r ON u.id = r.user_id WHERE u.enabled = 1 ORDER BY u.name`,
		nil,
	).Select(&users)
	return users, err
}

// DashboardTrendRow 仪表盘趋势行
type DashboardTrendRow struct {
	Label        string  `db:"label" json:"label"`
	Cost         float64 `db:"cost" json:"cost"`
	Count        int64   `db:"request_count" json:"request_count"`
	InputTokens  int64   `db:"input_tokens" json:"input_tokens"`
	OutputTokens int64   `db:"output_tokens" json:"output_tokens"`
	TotalTokens  int64   `db:"total_tokens" json:"total_tokens"`
	CachedTokens int64   `db:"cached_tokens" json:"cached_tokens"`
	CacheHitRate float64 `db:"cache_hit_rate" json:"cache_hit_rate"`
}

// DashboardSummary 仪表盘汇总指标
type DashboardSummary struct {
	UserCount     int64   `db:"user_count" json:"user_count"`
	ApiKeyCount   int64   `db:"api_key_count" json:"api_key_count"`
	TodayRequests int64   `db:"today_requests" json:"today_requests"`
	TodayCost     float64 `db:"today_cost" json:"today_cost"`
	TodayInput    int64   `json:"today_input"`
	TodayOutput   int64   `json:"today_output"`
	TodayTotal    int64   `json:"today_total"`
	TodayCached   int64   `json:"today_cached"`
	TodayCacheHit float64 `json:"today_cache_hit"`
}

// Dashboard 仪表盘：用户数/Key数/今日请求/今日费用 + token 指标 + 7 天费用趋势
func (us *UsageStore) Dashboard() (*DashboardSummary, []DashboardTrendRow, error) {
	var s DashboardSummary
	// 今日各项指标
	today := time.Now().Format("2006-01-02")
	todayStart := today + " 00:00:00"
	todayEnd := today + " 23:59:59"
	type todayRow struct {
		Requests int64   `db:"request_count"`
		Cost     float64 `db:"cost"`
		Input    int64   `db:"input_tokens"`
		Output   int64   `db:"output_tokens"`
		Total    int64   `db:"total_tokens"`
		Cached   int64   `db:"cached_tokens"`
	}
	var t todayRow
	err := us.s.Query(
		`SELECT COUNT(*) AS request_count, COALESCE(SUM(cost),0) AS cost,
			 COALESCE(SUM(input_tokens),0) AS input_tokens,
			 COALESCE(SUM(output_tokens),0) AS output_tokens,
			 COALESCE(SUM(total_tokens),0) AS total_tokens,
			 COALESCE(SUM(cached_tokens),0) AS cached_tokens
		 FROM usage_records WHERE created_at >= :start AND created_at <= :end`,
		map[string]any{"start": todayStart, "end": todayEnd},
	).Get(&t)
	if err != nil {
		return nil, nil, err
	}
	s.TodayRequests = t.Requests
	s.TodayCost = t.Cost
	s.TodayInput = t.Input
	s.TodayOutput = t.Output
	s.TodayTotal = t.Total
	s.TodayCached = t.Cached
	if t.Input > 0 {
		s.TodayCacheHit = float64(t.Cached) / float64(t.Input)
	}

	// 用户数、Key 数
	userCount, err := us.s.User().Count()
	if err != nil {
		return nil, nil, err
	}
	s.UserCount = userCount
	apiKeyCount, err := us.s.ApiKey().Count()
	if err != nil {
		return nil, nil, err
	}
	s.ApiKeyCount = apiKeyCount

	// 近 7 天趋势（含今天）
	var rows []DashboardTrendRow
	err = us.s.Query(
		`SELECT DATE(created_at) AS label, SUM(cost) AS cost, COUNT(*) AS request_count,
			 COALESCE(SUM(input_tokens),0) AS input_tokens,
			 COALESCE(SUM(output_tokens),0) AS output_tokens,
			 COALESCE(SUM(total_tokens),0) AS total_tokens,
			 COALESCE(SUM(cached_tokens),0) AS cached_tokens,
			 ROUND(CAST(SUM(cached_tokens) AS REAL) / NULLIF(SUM(input_tokens), 0), 4) AS cache_hit_rate
		 FROM usage_records
		 WHERE created_at >= :start
		 GROUP BY DATE(created_at)
		 ORDER BY label DESC`,
		map[string]any{"start": time.Now().AddDate(0, 0, -6).Format("2006-01-02") + " 00:00:00"},
	).Select(&rows)
	if err != nil {
		return nil, nil, err
	}
	return &s, rows, err
}
