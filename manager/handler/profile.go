package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"golang.org/x/crypto/bcrypt"
)

// ===== 请求/响应结构 =====

type UpdateProfileSelfReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateProfileSelfResp struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdatePasswordSelfReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ===== Handler =====

// UpdateProfileSelf 用户自助修改姓名和邮箱
func UpdateProfileSelf(ctx context.Context, req *UpdateProfileSelfReq) (*UpdateProfileSelfResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "name is required")
	}
	if len(name) > 64 {
		return nil, base.NewBizError(base.CodeInvalidParams, "name too long")
	}
	email := strings.TrimSpace(req.Email)
	if email != "" {
		if len(email) > 128 {
			return nil, base.NewBizError(base.CodeInvalidParams, "email too long")
		}
		if !strings.Contains(email, "@") {
			return nil, base.NewBizError(base.CodeInvalidParams, "invalid email format")
		}
	}
	if err := store.C().User().UpdateProfileSelf(cur.ID, name, email); err != nil {
		slog.Error("[Profile] UpdateProfileSelf failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &UpdateProfileSelfResp{Name: name, Email: email}, nil
}

// UpdatePasswordSelf 用户自助修改密码，需校验旧密码
func UpdatePasswordSelf(ctx context.Context, req *UpdatePasswordSelfReq) (*struct{}, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.OldPassword == "" || req.NewPassword == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "old_password and new_password are required")
	}
	if len(req.NewPassword) > 64 {
		return nil, base.NewBizError(base.CodeInvalidParams, "password too long")
	}

	// 重新查库拿当前密码哈希（CurrentUser 里的 user 不含 password）
	u, err := store.C().User().GetByIDAny(cur.ID)
	if err != nil || u == nil {
		slog.Error("[Profile] GetByIDAny failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	// 校验旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
		return nil, base.NewBizError(base.CodeWrongPassword, "old password incorrect")
	}

	// 新密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("[Profile] bcrypt failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if err := store.C().User().UpdatePassword(cur.ID, string(hash)); err != nil {
		slog.Error("[Profile] UpdatePassword failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	// 改密后吊销该用户所有登录会话（含当前），强制用新密码重登
	if err := sessionService.RevokeByUser(cur.ID); err != nil {
		slog.Error("[Profile] revoke sessions failed", "err", err, "user_id", cur.ID)
	}
	return &struct{}{}, nil
}
