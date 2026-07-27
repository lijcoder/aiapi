package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// Role 返回角色相关操作的命名空间。
func (s *Session) Role() *RoleStore {
	return &RoleStore{s: s}
}

// RoleStore 是角色相关操作的命名空间。
type RoleStore struct {
	s *Session
}

// Create 创建角色，回填 ID
func (rs *RoleStore) Create(r *model.Role) error {
	res, err := rs.s.Query(
		`INSERT INTO roles (name, code) VALUES (:name, :code)`,
		r,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// GetByName 按名称查询角色
func (rs *RoleStore) GetByName(name string) (*model.Role, error) {
	var r model.Role
	err := rs.s.Query(
		`SELECT * FROM roles WHERE name = :name`,
		map[string]any{"name": name},
	).Get(&r)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetByID 按 ID 查询角色
func (rs *RoleStore) GetByID(id int64) (*model.Role, error) {
	var r model.Role
	err := rs.s.Query(
		`SELECT * FROM roles WHERE id = :id`,
		map[string]any{"id": id},
	).Get(&r)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// List 列出全部角色
func (rs *RoleStore) List() ([]model.Role, error) {
	var rs_ []model.Role
	err := rs.s.Query(`SELECT * FROM roles ORDER BY id`, nil).Select(&rs_)
	if err != nil {
		return nil, err
	}
	return rs_, nil
}

// AssignRole 给用户分配角色
func (rs *RoleStore) AssignRole(userID, roleID int64) error {
	_, err := rs.s.Query(
		`INSERT OR IGNORE INTO user_roles (user_id, role_id) VALUES (:user_id, :role_id)`,
		map[string]any{"user_id": userID, "role_id": roleID},
	).Exec()
	return err
}

// RevokeRole 撤销用户的角色
func (rs *RoleStore) RevokeRole(userID, roleID int64) error {
	_, err := rs.s.Query(
		`DELETE FROM user_roles WHERE user_id = :user_id AND role_id = :role_id`,
		map[string]any{"user_id": userID, "role_id": roleID},
	).Exec()
	return err
}

// ListRolesByUser 查询用户拥有的全部角色
func (rs *RoleStore) ListRolesByUser(userID int64) ([]model.Role, error) {
	var roles []model.Role
	err := rs.s.Query(
		`SELECT r.* FROM roles r
		 INNER JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = :user_id
		 ORDER BY r.id`,
		map[string]any{"user_id": userID},
	).Select(&roles)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// ListRolesByUserIDs 批量查询多个用户的角色，返回 map[userID][]Role。
func (rs *RoleStore) ListRolesByUserIDs(userIDs []int64) (map[int64][]model.Role, error) {
	result := make(map[int64][]model.Role)
	if len(userIDs) == 0 {
		return result, nil
	}
	type row struct {
		UserID   int64  `db:"user_id"`
		RoleID   int64  `db:"id"`
		RoleName string `db:"name"`
		Code     string `db:"code"`
	}
	ph, args := inList("uid", userIDs)
	var rows []row
	err := rs.s.Query(
		`SELECT ur.user_id, r.id, r.name, r.code
		 FROM roles r
		 INNER JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id IN (`+ph+`)
		 ORDER BY r.id`,
		args,
	).Select(&rows)
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

// DeleteUserRoles 删除用户的全部角色关联。
func (rs *RoleStore) DeleteUserRoles(userID int64) error {
	_, err := rs.s.Query(
		`DELETE FROM user_roles WHERE user_id = :user_id`,
		map[string]any{"user_id": userID},
	).Exec()
	return err
}
