package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
)

// totpService TOTP 双因素认证业务。进程内单例，无状态。
var totpService = service.NewTOTPService()

// ===== 自助绑定 / 关闭（挂 Auth + Require）=====
//
// 注意：登录第二步 Login2FA 在 login.go 中——它与 Login/Refresh/Logout 同属
// 会话生命周期（签发 token、写 refresh cookie），所有 cookie 操作集中在 login.go。

type setup2FAResp struct {
	Secret      string `json:"secret"`       // Base32 密钥（手动录入备用）
	OtpauthURL  string `json:"otpauth_url"`  // otpauth:// 链接
	QRCode      string `json:"qr_code"`      // data:image/png;base64,... 二维码
	SetupTicket string `json:"setup_ticket"` // 绑定票据，确认时回传
	ExpiresIn   int    `json:"expires_in"`
}

// Setup2FASelf 生成 TOTP 密钥与二维码（不落库，确认后才生效）。
func Setup2FASelf(ctx context.Context) (*setup2FAResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	u, err := store.C().User().GetByIDAny(cur.ID)
	if err != nil || u == nil {
		slog.Error("[2FA] setup get user failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	if u.TotpSecret != "" {
		return nil, base.ErrBadReq("已开启两步验证，如需更换请先关闭")
	}
	setup, err := totpService.GenerateSetup(cur.ID, u.Account)
	if err != nil {
		slog.Error("[2FA] generate setup failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	return &setup2FAResp{
		Secret:      setup.Secret,
		OtpauthURL:  setup.OtpauthURL,
		QRCode:      setup.QRCodeData,
		SetupTicket: setup.SetupTicket,
		ExpiresIn:   setup.ExpiresIn,
	}, nil
}

type confirm2FAReq struct {
	SetupTicket string `json:"setup_ticket"`
	Code        string `json:"code"`
}

// Confirm2FASelf 校验首个验证码，通过则加密密钥落库，2FA 正式生效。
func Confirm2FASelf(ctx context.Context, req *confirm2FAReq) (*struct{}, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.SetupTicket == "" || req.Code == "" {
		return nil, base.ErrBadReq("验证码不能为空")
	}
	encSecret, err := totpService.ConfirmSetup(cur.ID, req.SetupTicket, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTOTPCodeWrong):
			return nil, base.ErrBadReq("验证码错误，请确认 Authenticator 时间同步后重试")
		case errors.Is(err, service.ErrTOTPInvalidTicket):
			return nil, base.ErrBadReq("绑定已过期，请重新生成二维码")
		default:
			slog.Error("[2FA] confirm setup failed", "err", err, "user_id", cur.ID)
			return nil, base.ErrInternal
		}
	}
	if err := store.C().User().UpdateTOTPSecret(cur.ID, encSecret); err != nil {
		slog.Error("[2FA] save totp secret failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}

type disable2FAReq struct {
	Password string `json:"password"`
}

// Disable2FASelf 关闭 2FA，需校验登录密码（防他人拿到已登录会话直接关 2FA）。
func Disable2FASelf(ctx context.Context, req *disable2FAReq) (*struct{}, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.Password == "" {
		return nil, base.ErrBadReq("密码不能为空")
	}
	u, err := store.C().User().GetByIDAny(cur.ID)
	if err != nil || u == nil {
		slog.Error("[2FA] disable get user failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	if !service.CheckPassword(u.Password, req.Password) {
		return nil, base.ErrBadReq("密码错误")
	}
	if err := store.C().User().UpdateTOTPSecret(cur.ID, ""); err != nil {
		slog.Error("[2FA] clear totp secret failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}
