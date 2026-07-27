package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
	"golang.org/x/crypto/bcrypt"
)

// sessionService 登录会话业务（签发/刷新/吊销）。进程内单例，无状态。
var sessionService = service.NewSessionService()

type loginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// Login 账号密码登录，成功签发 access JWT + refresh token（cookie）。
//
// 安全要点：
//   - 账号不存在/密码错/账号禁用统一返回相同文案，防账号枚举
//   - access JWT 短期（15min）放响应体，前端存内存
//   - refresh token 长期（滑动7天/绝对30天）放 HttpOnly cookie（HTTPS 下带 Secure 标记），JS 不可读
func Login(c echo.Context) error {
	var req loginReq
	if err := base.BindJSON(c, &req); err != nil {
		return base.Fail(c, base.CodeBadRequest, "请求体格式错误")
	}
	if req.Account == "" || req.Password == "" {
		return base.Fail(c, base.CodeBadRequest, "账号和密码不能为空")
	}

	user, err := store.C().User().GetByAccount(req.Account)
	if err != nil {
		slog.Error("login get user failed", "err", err)
		return base.Fail(c, base.ErrInternal.Code, base.ErrInternal.Msg)
	}
	// 统一错误文案：账号不存在/密码错/账号禁用均返回「账号或密码错误」
	// 防止攻击者通过差异响应枚举有效账号
	if user == nil || !user.Enabled {
		return base.Fail(c, base.CodeBadRequest, "账号或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return base.Fail(c, base.CodeBadRequest, "账号或密码错误")
	}

	// 签发 token 对
	pair, err := sessionService.Issue(user.ID, summarizeUA(c.Request().UserAgent()), c.RealIP())
	if err != nil {
		slog.Error("issue tokens failed", "err", err, "user_id", user.ID)
		return base.Fail(c, base.ErrInternal.Code, base.ErrInternal.Msg)
	}

	setRefreshCookie(c, pair.RefreshToken, pair.RefreshExp)

	return base.Ok(c, map[string]any{
		"access_token": pair.AccessToken,
		"expires_in":   int(base.AccessTTL.Seconds()),
	})
}

// Refresh 刷新 access JWT + 轮换 refresh token，含重用检测。
// 从 cookie 读 refresh token，不依赖 Authorization 头，因此不挂 Auth 中间件。
func Refresh(c echo.Context) error {
	cookie, err := c.Cookie(base.CookieName)
	if err != nil || cookie.Value == "" {
		return base.Fail(c, base.CodeUnauthorized, "未登录")
	}

	pair, err := sessionService.Refresh(cookie.Value)
	if err != nil {
		// 重用攻击 / 用户被禁用 → 清 cookie，强制重登
		if errors.Is(err, service.ErrSessionReuse) || errors.Is(err, service.ErrUserDisabled) {
			clearRefreshCookie(c)
			return base.Fail(c, base.CodeUnauthorized, "登录状态异常，请重新登录")
		}
		// token 过期/不存在 → 清 cookie
		if errors.Is(err, service.ErrSessionExpired) || errors.Is(err, service.ErrSessionNotFound) {
			clearRefreshCookie(c)
			return base.Fail(c, base.CodeUnauthorized, "登录已过期，请重新登录")
		}
		slog.Error("refresh tokens failed", "err", err)
		return base.Fail(c, base.ErrInternal.Code, base.ErrInternal.Msg)
	}

	setRefreshCookie(c, pair.RefreshToken, pair.RefreshExp)

	return base.Ok(c, map[string]any{
		"access_token": pair.AccessToken,
		"expires_in":   int(base.AccessTTL.Seconds()),
	})
}

// Logout 登出：吊销当前登录链（family）并清 cookie。
func Logout(c echo.Context) error {
	if cookie, err := c.Cookie(base.CookieName); err == nil && cookie.Value != "" {
		if err := sessionService.RevokeByRefreshToken(cookie.Value); err != nil {
			slog.Error("logout revoke session failed", "err", err)
		}
	}
	clearRefreshCookie(c)
	return base.Ok(c, nil)
}

// ===== 辅助函数 =====

// setRefreshCookie 写 refresh token 到 HttpOnly cookie。
// Path 限定 /manager，SameSite=Lax 防 CSRF。
// Secure 跟随当前请求是否 HTTPS：HTTPS 部署下开启，HTTP（如本地开发）下关闭，
// 否则浏览器会拒绝存储/发送 Secure cookie，导致 refresh 失败、登录态丢失。
func setRefreshCookie(c echo.Context, token string, expires time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     base.CookieName,
		Value:    token,
		Path:     base.CookiePath,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     base.CookieName,
		Value:    "",
		Path:     base.CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

// summarizeUA 截断 User-Agent，防止过长字符串入库。
func summarizeUA(ua string) string {
	ua = strings.TrimSpace(ua)
	if len(ua) > 200 {
		return ua[:200]
	}
	return ua
}
