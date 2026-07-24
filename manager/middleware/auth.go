package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/manager/service"
)

// sessionService 与 handler 共用同一实例（无状态）。
var sessionService = service.NewSessionService()

// fail 输出未登录响应。不清 cookie：401 本身即指示客户端重新登录，
// 且 fail 也用于 DB 瞬时错误等场景，误清好 cookie 会强制重登。
func fail(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, constant.HttpGeneralResp{
		Code: base.CodeUnauthorized.ID,
		Msg:  "unauthorized",
	})
}

// failExpired 输出 access 过期响应，前端据此触发 /refresh。
func failExpired(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, constant.HttpGeneralResp{
		Code: base.CodeTokenExpired.ID,
		Msg:  "token expired",
	})
}

// Auth 校验 access JWT：从 Authorization: Bearer <token> 读 → 解析 → 加载用户 → 注入 context。
// 权限校验由 Require 中间件负责。
func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return fail(c)
		}
		token, ok := parseBearer(auth)
		if !ok {
			return fail(c)
		}

		user, err := sessionService.ValidateAccess(token)
		if err != nil {
			// access 过期 → 前端可调 /refresh 续期
			if errors.Is(err, service.ErrTokenExpired) {
				return failExpired(c)
			}
			// 用户被禁用/删除 → 吊销其所有 session，强制重登
			if errors.Is(err, service.ErrUserDisabled) {
				return fail(c)
			}
			// 签名错/格式错/issuer 不符 → 视为未登录
			return fail(c)
		}

		c.SetRequest(c.Request().WithContext(base.WithAuth(c.Request().Context(), user, nil)))
		return next(c)
	}
}

// parseBearer 从 "Bearer <token>" 中提取 token。
func parseBearer(auth string) (string, bool) {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
		return "", false
	}
	return auth[len(prefix):], true
}
