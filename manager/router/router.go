package router

import (
	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/manager/handler"
	"github.com/lijcoder/aiapi/manager/middleware"
)

// Register 在已建好的 /manager group 上注册后台管理子路由。
// 所有接口均为 POST，参数走 JSON body。
//   - login 不挂中间件
//   - logout 挂 Auth（需登录态），不挂 Require（无需接口授权）
//   - 其余挂 Auth + Require（需登录态 + 接口授权）
func Register(g *echo.Group) {
	g.POST("/login", handler.Login)

	g.Use(middleware.Auth)
	g.POST("/logout", handler.Logout)

	g.Use(middleware.Require)
	g.POST("/self", base.Wrap(handler.Self))
	g.POST("/recharge/self", base.Wrap(handler.RechargeSelf))
	g.POST("/recharge", base.Wrap(handler.RechargeAdmin))
	g.POST("/recharge/records", base.Wrap(handler.RechargeRecords))
	g.POST("/recharge/records/self", base.Wrap(handler.RechargeSelfRecords))

	g.POST("/models", base.Wrap(handler.ListModels))

	g.POST("/apikeys/list/self", base.Wrap(handler.ListApiKeySelf))
	g.POST("/apikeys/create/self", base.Wrap(handler.CreateApiKeySelf))
	g.POST("/apikeys/toggle/self", base.Wrap(handler.ToggleApiKeySelf))
	g.POST("/apikeys/delete/self", base.Wrap(handler.DeleteApiKeySelf))
	g.POST("/apikeys/rename/self", base.Wrap(handler.RenameApiKeySelf))
	g.POST("/apikeys/budget/self", base.Wrap(handler.UpdateBudgetApiKeySelf))

	g.POST("/usage/stats/self", base.Wrap(handler.UsageStatsSelf))
	g.POST("/usage/filters/self", base.Wrap(handler.UsageFiltersSelf))
}
