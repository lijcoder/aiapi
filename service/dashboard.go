package service

import (
	"github.com/lijcoder/aiapi/store"
)

// DashboardSummary 仪表盘汇总指标。
type DashboardSummary struct {
	UserCount     int64   `json:"user_count"`
	ApiKeyCount   int64   `json:"api_key_count"`
	TodayRequests int64   `json:"today_requests"`
	TodayCost     float64 `json:"today_cost"`
	TodayInput    int64   `json:"today_input"`
	TodayOutput   int64   `json:"today_output"`
	TodayTotal    int64   `json:"today_total"`
	TodayCached   int64   `json:"today_cached"`
	TodayCacheHit float64 `json:"today_cache_hit"`
}

// DashboardService 封装仪表盘业务逻辑（跨 Store 组装）。
type DashboardService struct{}

// NewDashboardService 创建 DashboardService。
func NewDashboardService() *DashboardService { return &DashboardService{} }

// Dashboard 仪表盘：用户数/Key数/今日请求/今日费用 + token 指标 + 7 天趋势。
func (s *DashboardService) Dashboard() (*DashboardSummary, []store.DashboardTrendRow, error) {
	t, err := store.C().Usage().TodayMetrics()
	if err != nil {
		return nil, nil, err
	}

	userCount, err := store.C().User().Count()
	if err != nil {
		return nil, nil, err
	}
	apiKeyCount, err := store.C().ApiKey().Count()
	if err != nil {
		return nil, nil, err
	}

	trend, err := store.C().Usage().Trend7d()
	if err != nil {
		return nil, nil, err
	}

	summary := &DashboardSummary{
		UserCount:     userCount,
		ApiKeyCount:   apiKeyCount,
		TodayRequests: t.Requests,
		TodayCost:     t.Cost,
		TodayInput:    t.Input,
		TodayOutput:   t.Output,
		TodayTotal:    t.Total,
		TodayCached:   t.Cached,
	}
	if t.Input > 0 {
		summary.TodayCacheHit = float64(t.Cached) / float64(t.Input)
	}
	return summary, trend, nil
}
