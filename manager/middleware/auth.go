package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
)

// fail 输出未登录响应。不清 cookie：401 本身即指示客户端重新登录，
// 且 fail 也用于 DB 瞬时错误等场景，误清好 cookie 会强制重登。
func fail(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, constant.HttpGeneralResp{
		Code: base.CodeUnauthorized.ID,
		Msg:  "unauthorized",
	})
}

// Auth 校验登录态：从 cookie 读 token → 加载 session/user → 注入到请求 context。
// 权限校验由 Require 中间件负责。
func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(base.CookieName)
		if err != nil || cookie.Value == "" {
			return fail(c)
		}

		sess, err := store.C().UserSession().GetByToken(cookie.Value)
		if err != nil {
			return fail(c)
		}
		if sess == nil {
			return fail(c)
		}
		if sess.ExpiresAt.Before(time.Now()) {
			_ = store.C().UserSession().Delete(cookie.Value)
			return fail(c)
		}

		user, err := store.C().User().GetByID(sess.UserID)
		if err != nil {
			return fail(c)
		}
		if user == nil || !user.Enabled {
			_ = store.C().UserSession().Delete(cookie.Value)
			return fail(c)
		}

		c.SetRequest(c.Request().WithContext(base.WithAuth(c.Request().Context(), user, nil)))
		return next(c)
	}
}
