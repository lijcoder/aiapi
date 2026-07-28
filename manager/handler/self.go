package handler

import (
	"context"
	"log/slog"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// userInfo 用户基本信息
type userInfo struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Account     string  `json:"account"`
	Email       string  `json:"email"`
	Budget      float64 `json:"budget"`
	TotpEnabled bool    `json:"totp_enabled"` // 是否已开启两步验证
}

// menuItem 菜单项（含子菜单）
type menuItem struct {
	ID       int64       `json:"id"`
	ParentID int64       `json:"parent_id"`
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Icon     string      `json:"icon,omitempty"`
	Children []*menuItem `json:"children,omitempty"`
}

// SelfResp /manager/self 响应
type SelfResp struct {
	User        userInfo               `json:"user"`
	Menus       []*menuItem            `json:"menus"`
	Permissions []model.RolePermission `json:"permissions"`
}

// Self 返回当前登录用户的实时信息（从数据库查询） + 菜单树 + 权限列表。
func Self(ctx context.Context) (*SelfResp, *base.BizError) {
	curID := base.CurrentUser(ctx).ID
	user, err := store.C().User().GetByID(curID)
	if err != nil {
		return nil, base.ErrInternal
	}
	if user == nil {
		return nil, base.ErrInternal
	}

	// 查用户角色 ID 列表
	roles, err := store.C().Role().ListRolesByUser(curID)
	if err != nil {
		slog.Error("[Self] ListRolesByUser failed", "err", err, "user_id", curID)
		return nil, base.ErrInternal
	}
	var roleIDs []int64
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}

	// 按角色查菜单
	menus, err := store.C().Menu().ListByRoleIDs(roleIDs)
	if err != nil {
		slog.Error("[Self] ListByRoleIDs failed", "err", err, "role_ids", roleIDs)
		return nil, base.ErrInternal
	}

	return &SelfResp{
		User: userInfo{
			ID:          user.ID,
			Name:        user.Name,
			Account:     user.Account,
			Email:       user.Email,
			Budget:      user.Budget,
			TotpEnabled: user.TotpSecret != "",
		},
		Menus:       buildMenuTree(menus),
		Permissions: base.CurrentPermissions(ctx),
	}, nil
}

// buildMenuTree 将扁平菜单列表组装为树形结构（按 sort_order 已排序）。
func buildMenuTree(menus []model.Menu) []*menuItem {
	itemMap := make(map[int64]*menuItem, len(menus))
	for _, m := range menus {
		itemMap[m.ID] = &menuItem{
			ID:       m.ID,
			ParentID: m.ParentID,
			Name:     m.Name,
			Path:     m.Path,
			Icon:     m.Icon,
		}
	}
	var roots []*menuItem
	for _, m := range menus {
		item := itemMap[m.ID]
		if m.ParentID == 0 {
			roots = append(roots, item)
		} else if parent, ok := itemMap[m.ParentID]; ok {
			parent.Children = append(parent.Children, item)
		}
	}
	return roots
}
