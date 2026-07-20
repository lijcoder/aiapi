package base

import (
	"context"

	"github.com/lijcoder/aiapi/store/model"
)

// ===== 请求级登录态：通过 context.Context 携带 =====
// 每个请求有独立的 context 树，context.WithValue 返回新 context（不可变），
// 并发请求各自持有自己的 context，不存在共享可变状态，读取并发安全。

type ctxKey int

const (
	keyUser  ctxKey = 1
	keyPerms ctxKey = 2
)

// WithAuth 把当前登录用户与权限注入 ctx，返回新 context（不可变）。
func WithAuth(ctx context.Context, user *model.User, perms []model.RolePermission) context.Context {
	ctx = context.WithValue(ctx, keyUser, user)
	ctx = context.WithValue(ctx, keyPerms, perms)
	return ctx
}

// CurrentUser 从 ctx 取当前登录用户；不存在返回 nil
func CurrentUser(ctx context.Context) *model.User {
	v, _ := ctx.Value(keyUser).(*model.User)
	return v
}

// WithPerms 把权限列表注入 ctx，返回新 context。
func WithPerms(ctx context.Context, perms []model.RolePermission) context.Context {
	return context.WithValue(ctx, keyPerms, perms)
}

// CurrentPermissions 从 ctx 取当前登录用户的权限列表
func CurrentPermissions(ctx context.Context) []model.RolePermission {
	v, _ := ctx.Value(keyPerms).([]model.RolePermission)
	return v
}
