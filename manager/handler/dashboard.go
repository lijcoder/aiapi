package handler

import (
	"context"
	"log/slog"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
)

// DashboardResp 仪表盘响应
type DashboardResp struct {
	Summary service.DashboardSummary   `json:"summary"`
	Trend   []store.DashboardTrendRow `json:"trend"`
}

// Dashboard 仪表盘：用户数/Key数/今日请求/今日费用 + 7天费用趋势
func Dashboard(ctx context.Context) (*DashboardResp, *base.BizError) {
	summary, trend, err := service.NewDashboardService().Dashboard()
	if err != nil {
		slog.Error("[Dashboard] failed", "err", err)
		return nil, base.ErrInternal
	}
	if trend == nil {
		trend = []store.DashboardTrendRow{}
	}
	return &DashboardResp{Summary: *summary, Trend: trend}, nil
}
