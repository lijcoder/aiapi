package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
)

// forbidden 输出权限不足响应
func forbidden(c echo.Context) error {
	return c.JSON(http.StatusForbidden, constant.HttpGeneralResp{
		Code: base.CodeForbidden.ID,
		Msg:  "没有操作权限",
	})
}

// Require 接口级权限校验：加载用户权限，检查当前请求路径是否命中授权表，否则 403。
// 应在 Auth 中间件之后使用——Auth 已把当前用户注入请求 context。
func Require(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := base.CurrentUser(c.Request().Context())
		if user == nil {
			return fail(c)
		}
		perms, err := store.C().RolePermission().ListByUser(user.ID)
		if err != nil {
			return fail(c)
		}
		if err := base.CheckPathPermission(perms, c.Path()); err != nil {
			return forbidden(c)
		}
		c.SetRequest(c.Request().WithContext(base.WithPerms(c.Request().Context(), perms)))
		return next(c)
	}
}
