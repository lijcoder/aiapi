package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// Login 账号密码登录，成功签发 session 并写入 cookie
func Login(c echo.Context) error {
	var req loginReq
	if err := base.BindJSON(c, &req); err != nil {
		return base.Fail(c, base.CodeBadRequest, "invalid request body")
	}
	if req.Account == "" || req.Password == "" {
		return base.Fail(c, base.CodeInvalidParams, "account and password are required")
	}

	user, err := store.C().User().GetByAccount(req.Account)
	if err != nil {
		return base.Fail(c, base.CodeUnknown, base.InternalServerError)
	}
	if user == nil {
		return base.Fail(c, base.CodeWrongPassword, "account or password incorrect")
	}
	if !user.Enabled {
		return base.Fail(c, base.CodeUserDisabled, "user disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return base.Fail(c, base.CodeWrongPassword, "account or password incorrect")
	}

	token, err := newToken()
	if err != nil {
		return base.Fail(c, base.CodeUnknown, base.InternalServerError)
	}
	expiresAt := time.Now().Add(base.SessionTTL)
	if err := store.C().UserSession().Create(token, user.ID, expiresAt); err != nil {
		return base.Fail(c, base.CodeUnknown, base.InternalServerError)
	}

	c.SetCookie(&http.Cookie{
		Name:     base.CookieName,
		Value:    token,
		Path:     base.CookiePath,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return base.Ok(c, nil)
}

// Logout 登出，删除 session 并清 cookie
func Logout(c echo.Context) error {
	if cookie, err := c.Cookie(base.CookieName); err == nil && cookie.Value != "" {
		if err := store.C().UserSession().Delete(cookie.Value); err != nil {
			slog.Error("logout delete session failed", "err", err)
		}
	}
	c.SetCookie(&http.Cookie{
		Name:     base.CookieName,
		Value:    "",
		Path:     base.CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return base.Ok(c, nil)
}

func newToken() (string, error) {
	b := make([]byte, base.TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
