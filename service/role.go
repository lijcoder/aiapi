package service

import (
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// RoleService 封装角色相关业务逻辑（事务编排）。
type RoleService struct{}

// NewRoleService 创建 RoleService。
func NewRoleService() *RoleService { return &RoleService{} }

// ReplaceUserRoles 全量替换用户的角色关联：删旧 → 循环插新，事务内执行。
func (s *RoleService) ReplaceUserRoles(userID int64, roleIDs []int64) error {
	return store.C().T(func(ss *store.Session) error {
		if err := ss.Role().DeleteUserRoles(userID); err != nil {
			return err
		}
		for _, rid := range roleIDs {
			if err := ss.Role().AssignRole(userID, rid); err != nil {
				return err
			}
		}
		return nil
	})
}

// RolesByUserIDs 批量查询多个用户的角色，组装为 map[userID][]Role。
func (s *RoleService) RolesByUserIDs(userIDs []int64) (map[int64][]model.Role, error) {
	result := make(map[int64][]model.Role)
	rows, err := store.C().Role().ListRoleRowsByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.UserID] = append(result[r.UserID], model.Role{
			ID:   r.RoleID,
			Name: r.RoleName,
			Code: r.Code,
		})
	}
	return result, nil
}
