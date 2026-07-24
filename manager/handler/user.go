package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
	"golang.org/x/crypto/bcrypt"
)

// ===== 请求/响应结构 =====

type userItem struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Account   string  `json:"account"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
	Enabled   bool    `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	RoleNames []string `json:"role_names"`
	RoleIDs   []int64  `json:"role_ids"`
}

type ListUsersReq struct {
	Keyword string `json:"keyword"` // 按姓名/账号模糊搜索
}

type ListUsersResp struct {
	Users []userItem `json:"users"`
}

type CreateUserReq struct {
	Name      string  `json:"name"`
	Account   string  `json:"account"`
	Password  string  `json:"password"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
}

type UpdateUserReq struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
}

type UserIdReq struct {
	ID int64 `json:"id"`
}

type ResetPasswordReq struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

type AssignRolesReq struct {
	ID      int64   `json:"id"`
	RoleIDs []int64 `json:"role_ids"`
}

type ListRolesResp struct {
	Roles []roleItem `json:"roles"`
}

type roleItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

// ListRoles 查询全部角色（用于分配角色弹窗）
func ListRoles(ctx context.Context) (*ListRolesResp, *base.BizError) {
	roles, err := store.C().Role().List()
	if err != nil {
		slog.Error("[Role] List failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	items := make([]roleItem, 0, len(roles))
	for _, r := range roles {
		items = append(items, roleItem{
			ID:   r.ID,
			Name: r.Name,
			Code: r.Code,
		})
	}
	return &ListRolesResp{Roles: items}, nil
}

// ===== Handler =====

// ListUsers 管理员查询用户列表（含角色信息）
func ListUsers(ctx context.Context, req *ListUsersReq) (*ListUsersResp, *base.BizError) {
	users, err := store.C().User().List()
	if err != nil {
		slog.Error("[User] List failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	// 批量查角色
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	roleMap, err := store.C().Role().ListRolesByUserIDs(ids)
	if err != nil {
		slog.Error("[User] ListRolesByUserIDs failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	// 关键字过滤
	items := make([]userItem, 0, len(users))
	for _, u := range users {
		if req.Keyword != "" {
			kw := req.Keyword
		if !containsFold(u.Name, kw) && !containsFold(u.Account, kw) {
				continue
			}
		}
		roles := roleMap[u.ID]
		roleNames := make([]string, 0, len(roles))
		roleIDs := make([]int64, 0, len(roles))
		for _, r := range roles {
			roleNames = append(roleNames, r.Name)
			roleIDs = append(roleIDs, r.ID)
		}
		items = append(items, userItem{
			ID:        u.ID,
			Name:      u.Name,
			Account:   u.Account,
			Budget:    u.Budget,
			Unlimited: u.Unlimited,
			Enabled:   u.Enabled,
			CreatedAt: u.CreatedAt,
			RoleNames: roleNames,
			RoleIDs:   roleIDs,
		})
	}
	return &ListUsersResp{Users: items}, nil
}

// CreateUser 管理员创建用户
func CreateUser(ctx context.Context, req *CreateUserReq) (*userItem, *base.BizError) {
	if req.Name == "" || req.Account == "" || req.Password == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "name, account, password are required")
	}

	// 检查账号唯一性
	exist, err := store.C().User().GetByAccount(req.Account)
	if err != nil {
		slog.Error("[User] GetByAccount failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if exist != nil {
		return nil, base.NewBizError(base.CodeAccountExists, "account already exists")
	}

	// bcrypt 加密
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("[User] bcrypt failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	u := &model.User{
		Name:      req.Name,
		Account:   req.Account,
		Password:  string(hash),
		Budget:    req.Budget,
		Unlimited: req.Unlimited,
		Enabled:   true,
	}
	if err := store.C().User().Create(u); err != nil {
		if store.IsUniqueConstraintErr(err) {
			return nil, base.NewBizError(base.CodeAccountExists, "account already exists")
		}
		slog.Error("[User] Create failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	return &userItem{
		ID:        u.ID,
		Name:      u.Name,
		Account:   u.Account,
		Budget:    u.Budget,
		Unlimited: u.Unlimited,
		Enabled:   true,
		CreatedAt: u.CreatedAt,
		RoleNames: []string{},
		RoleIDs:   []int64{},
	}, nil
}

// UpdateUser 管理员编辑用户基本信息（账号不可改）
func UpdateUser(ctx context.Context, req *UpdateUserReq) (*userItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	if req.Name == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "name is required")
	}

	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if u == nil {
		return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
	}

	u.Name = req.Name
	u.Budget = req.Budget
	u.Unlimited = req.Unlimited
	if err := store.C().User().UpdateProfile(u); err != nil {
		slog.Error("[User] UpdateProfile failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}

	// 回查角色
	roles, _ := store.C().Role().ListRolesByUser(u.ID)
	roleNames := make([]string, 0, len(roles))
	roleIDs := make([]int64, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
		roleIDs = append(roleIDs, r.ID)
	}

	return &userItem{
		ID:        u.ID,
		Name:      u.Name,
		Account:   u.Account,
		Budget:    u.Budget,
		Unlimited: u.Unlimited,
		Enabled:   u.Enabled,
		CreatedAt: u.CreatedAt,
		RoleNames: roleNames,
		RoleIDs:   roleIDs,
	}, nil
}

// ToggleUser 启用/禁用用户
func ToggleUser(ctx context.Context, req *UserIdReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if u == nil {
		return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
	}
	newEnabled := !u.Enabled
	if err := store.C().User().SetEnabled(req.ID, newEnabled); err != nil {
		slog.Error("[User] SetEnabled failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	// 禁用用户时吊销其所有登录会话，强制立即下线
	if !newEnabled {
		if err := sessionService.RevokeByUser(req.ID); err != nil {
			slog.Error("[User] revoke sessions failed", "err", err, "id", req.ID)
		}
	}
	return &struct{}{}, nil
}

// ResetPassword 管理员重置用户密码
func ResetPassword(ctx context.Context, req *ResetPasswordReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	if req.Password == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "password is required")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if u == nil {
		return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("[User] bcrypt failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if err := store.C().User().UpdatePassword(req.ID, string(hash)); err != nil {
		slog.Error("[User] UpdatePassword failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	// 重置密码后吊销该用户所有登录会话，必须用新密码重登
	if err := sessionService.RevokeByUser(req.ID); err != nil {
		slog.Error("[User] revoke sessions failed", "err", err, "id", req.ID)
	}
	return &struct{}{}, nil
}

// AssignRoles 管理员给用户分配角色（全量替换）
func AssignRoles(ctx context.Context, req *AssignRolesReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if u == nil {
		return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
	}
	if err := store.C().Role().ReplaceUserRoles(req.ID, req.RoleIDs); err != nil {
		slog.Error("[User] ReplaceUserRoles failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &struct{}{}, nil
}

// ===== 工具函数 =====

// containsFold 大小写不敏感的子串匹配
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}