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

	// 超管：用户管理
	g.POST("/users/list", base.Wrap(handler.ListUsers))
	g.POST("/users/create", base.Wrap(handler.CreateUser))
	g.POST("/users/update", base.Wrap(handler.UpdateUser))
	g.POST("/users/toggle", base.Wrap(handler.ToggleUser))
	g.POST("/users/reset-password", base.Wrap(handler.ResetPassword))
	g.POST("/users/assign-roles", base.Wrap(handler.AssignRoles))
	g.POST("/roles/list", base.Wrap(handler.ListRoles))

	// 超管：Provider 管理
	g.POST("/providers/list", base.Wrap(handler.ListProviders))
	g.POST("/providers/create", base.Wrap(handler.CreateProvider))
	g.POST("/providers/update", base.Wrap(handler.UpdateProvider))
	g.POST("/providers/toggle", base.Wrap(handler.ToggleProvider))

	// 超管：模型定价管理
	g.POST("/models/list", base.Wrap(handler.ListModelsAdmin))
	g.POST("/models/create", base.Wrap(handler.CreateModel))
	g.POST("/models/update", base.Wrap(handler.UpdateModel))
	g.POST("/models/delete", base.Wrap(handler.DeleteModel))

	// 超管：管理指定用户的 API Key
	g.POST("/apikeys/list", base.Wrap(handler.ListApiKeyAdmin))
	g.POST("/apikeys/toggle", base.Wrap(handler.ToggleApiKeyAdmin))
	g.POST("/apikeys/delete", base.Wrap(handler.DeleteApiKeyAdmin))
	g.POST("/apikeys/rename", base.Wrap(handler.RenameApiKeyAdmin))
	g.POST("/apikeys/budget", base.Wrap(handler.UpdateBudgetApiKeyAdmin))

	// 超管：全平台充值流水
	g.POST("/recharge/records/list", base.Wrap(handler.ListRechargeRecords))

	// 超管：全局统计
	g.POST("/usage/stats", base.Wrap(handler.UsageStatsAdmin))
	g.POST("/usage/filters", base.Wrap(handler.UsageFiltersAdmin))
}
