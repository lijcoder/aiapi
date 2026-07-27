package service

import "github.com/lijcoder/aiapi/store"

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
