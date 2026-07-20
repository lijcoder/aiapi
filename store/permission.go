package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// RolePermission 返回权限相关操作的命名空间。
func (s *Session) RolePermission() *RolePermissionStore {
	return &RolePermissionStore{s: s}
}

// RolePermissionStore 是接口级权限相关操作的命名空间。
type RolePermissionStore struct {
	s *Session
}

// Grant 为角色授予一条权限 (entity, action, value)
func (ps *RolePermissionStore) Grant(roleID int64, entity, action, value string) error {
	_, err := ps.s.Query(
		`INSERT OR IGNORE INTO role_permission (role_id, entity, action, value)
		 VALUES (:role_id, :entity, :action, :value)`,
		map[string]any{
			"role_id": roleID,
			"entity":  entity,
			"action":  action,
			"value":   value,
		},
	).Exec()
	return err
}

// Revoke 撤销角色的一条权限
func (ps *RolePermissionStore) Revoke(roleID int64, entity, action, value string) error {
	_, err := ps.s.Query(
		`DELETE FROM role_permission
		 WHERE role_id = :role_id AND entity = :entity AND action = :action AND value = :value`,
		map[string]any{
			"role_id": roleID,
			"entity":  entity,
			"action":  action,
			"value":   value,
		},
	).Exec()
	return err
}

// ListByRole 查询角色的全部权限
func (ps *RolePermissionStore) ListByRole(roleID int64) ([]model.RolePermission, error) {
	var perms []model.RolePermission
	err := ps.s.Query(
		`SELECT * FROM role_permission WHERE role_id = :role_id ORDER BY id`,
		map[string]any{"role_id": roleID},
	).Select(&perms)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// ListByUser 查询用户经由其所有角色拥有的全部权限（去重）
func (ps *RolePermissionStore) ListByUser(userID int64) ([]model.RolePermission, error) {
	var perms []model.RolePermission
	err := ps.s.Query(
		`SELECT DISTINCT rp.* FROM role_permission rp
		 INNER JOIN user_roles ur ON ur.role_id = rp.role_id
		 WHERE ur.user_id = :user_id
		 ORDER BY rp.id`,
		map[string]any{"user_id": userID},
	).Select(&perms)
	if err != nil {
		return nil, err
	}
	return perms, nil
}
