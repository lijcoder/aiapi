package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ===== 请求/响应结构 =====

type userItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Account   string    `json:"account"`
	Budget    float64   `json:"budget"`
	Unlimited bool      `json:"unlimited"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	RoleNames []string  `json:"role_names"`
	RoleIDs   []int64   `json:"role_ids"`
}

type ListUsersReq struct {
	Keyword string `json:"keyword"` // 按姓名/账号模糊搜索
	base.PageReq
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
		return nil, base.ErrInternal
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

// ListUsers 管理员查询用户列表（含角色信息，分页）
func ListUsers(ctx context.Context, req *ListUsersReq) (*base.PageResult[userItem], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	users, err := store.C().SetPage(pc).User().List(strings.TrimSpace(req.Keyword))
	if err != nil {
		slog.Error("[User] List failed", "err", err)
		return nil, base.ErrInternal
	}

	// 批量查角色
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	roleMap, err := service.NewRoleService().RolesByUserIDs(ids)
	if err != nil {
		slog.Error("[User] ListRolesByUserIDs failed", "err", err)
		return nil, base.ErrInternal
	}

	items := make([]userItem, 0, len(users))
	for _, u := range users {
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
	return &base.PageResult[userItem]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// GetUserAdmin 超管查询单个用户信息
func GetUserAdmin(ctx context.Context, req *UserIdReq) (*userItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	u, err := store.C().User().GetByID(req.ID)
	if err != nil {
		slog.Error("[User] GetByID failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	return &userItem{
		ID:        u.ID,
		Name:      u.Name,
		Account:   u.Account,
		Budget:    u.Budget,
		Unlimited: u.Unlimited,
		Enabled:   u.Enabled,
		CreatedAt: u.CreatedAt,
	}, nil
}

// CreateUser 管理员创建用户
func CreateUser(ctx context.Context, req *CreateUserReq) (*userItem, *base.BizError) {
	if req.Name == "" || req.Account == "" || req.Password == "" {
		return nil, base.ErrBadReq("名称、账号、密码不能为空")
	}

	// 检查账号唯一性
	exist, err := store.C().User().GetByAccount(req.Account)
	if err != nil {
		slog.Error("[User] GetByAccount failed", "err", err)
		return nil, base.ErrInternal
	}
	if exist != nil {
		return nil, base.ErrBadReq("账号已存在")
	}

	// 密码哈希（统一走 service 收口）
	hash, err := service.HashPassword(req.Password)
	if err != nil {
		slog.Error("[User] hash password failed", "err", err)
		return nil, base.ErrInternal
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
			return nil, base.ErrBadReq("账号已存在")
		}
		slog.Error("[User] Create failed", "err", err)
		return nil, base.ErrInternal
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
		return nil, base.ErrBadReq("id 不能为空")
	}
	if req.Name == "" {
		return nil, base.ErrBadReq("名称不能为空")
	}

	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if u == nil {
		return nil, base.ErrNotFound("用户不存在")
	}

	u.Name = req.Name
	u.Budget = req.Budget
	u.Unlimited = req.Unlimited
	if err := store.C().User().UpdateProfile(u); err != nil {
		slog.Error("[User] UpdateProfile failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
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
		return nil, base.ErrBadReq("id 不能为空")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if u == nil {
		return nil, base.ErrNotFound("用户不存在")
	}
	newEnabled := !u.Enabled
	if err := store.C().User().SetEnabled(req.ID, newEnabled); err != nil {
		slog.Error("[User] SetEnabled failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
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
		return nil, base.ErrBadReq("id 不能为空")
	}
	if req.Password == "" {
		return nil, base.ErrBadReq("密码不能为空")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if u == nil {
		return nil, base.ErrNotFound("用户不存在")
	}
	hash, err := service.HashPassword(req.Password)
	if err != nil {
		slog.Error("[User] hash password failed", "err", err)
		return nil, base.ErrInternal
	}
	if err := store.C().User().UpdatePassword(req.ID, hash); err != nil {
		slog.Error("[User] UpdatePassword failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
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
		return nil, base.ErrBadReq("id 不能为空")
	}
	u, err := store.C().User().GetByIDAny(req.ID)
	if err != nil {
		slog.Error("[User] GetByIDAny failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if u == nil {
		return nil, base.ErrNotFound("用户不存在")
	}
	if err := service.NewRoleService().ReplaceUserRoles(req.ID, req.RoleIDs); err != nil {
		slog.Error("[User] ReplaceUserRoles failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}
