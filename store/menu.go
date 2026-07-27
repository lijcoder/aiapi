package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lijcoder/aiapi/store/model"
)

// Menu 返回菜单相关操作的命名空间。
func (s *Session) Menu() *MenuStore {
	return &MenuStore{s: s}
}

// MenuStore 是菜单相关操作的命名空间，包含菜单 CRUD 和角色-菜单关联。
type MenuStore struct {
	s *Session
}

// ===== 菜单 CRUD =====

// ListByRoleIDs 按角色 ID 列表查询菜单（DISTINCT 去重，按 sort_order 排序）。
func (ms *MenuStore) ListByRoleIDs(roleIDs []int64) ([]model.Menu, error) {
	if len(roleIDs) == 0 {
		return []model.Menu{}, nil
	}
	ph, args := inList("rid", roleIDs)
	var menus []model.Menu
	err := ms.s.Query(
		`SELECT DISTINCT m.* FROM menus m
		 JOIN role_menus rm ON m.id = rm.menu_id
		 WHERE rm.role_id IN (`+ph+`)
		 ORDER BY m.sort_order`,
		args,
	).Select(&menus)
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// ListAll 查询全部菜单（按 sort_order 排序）。
func (ms *MenuStore) ListAll() ([]model.Menu, error) {
	var menus []model.Menu
	err := ms.s.Query(
		`SELECT * FROM menus ORDER BY sort_order`,
		nil,
	).Select(&menus)
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// GetByID 按 ID 查询菜单。
func (ms *MenuStore) GetByID(id int64) (*model.Menu, error) {
	var m model.Menu
	err := ms.s.Query(
		`SELECT * FROM menus WHERE id = :id`,
		map[string]any{"id": id},
	).Get(&m)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 新增菜单。
func (ms *MenuStore) Create(m *model.Menu) error {
	_, err := ms.s.Query(
		`INSERT INTO menus (parent_id, name, path, icon, sort_order)
		 VALUES (:parent_id, :name, :path, :icon, :sort_order)`,
		map[string]any{
			"parent_id":  m.ParentID,
			"name":       m.Name,
			"path":       m.Path,
			"icon":       m.Icon,
			"sort_order": m.SortOrder,
		},
	).Exec()
	return err
}

// Update 更新菜单。
func (ms *MenuStore) Update(m *model.Menu) error {
	_, err := ms.s.Query(
		`UPDATE menus SET parent_id=:parent_id, name=:name, path=:path, icon=:icon, sort_order=:sort_order
		 WHERE id=:id`,
		map[string]any{
			"id":         m.ID,
			"parent_id":  m.ParentID,
			"name":       m.Name,
			"path":       m.Path,
			"icon":       m.Icon,
			"sort_order": m.SortOrder,
		},
	).Exec()
	return err
}

// Delete 删除菜单。
func (ms *MenuStore) Delete(id int64) error {
	_, err := ms.s.Query(
		`DELETE FROM menus WHERE id = :id`,
		map[string]any{"id": id},
	).Exec()
	return err
}

// ===== 角色-菜单关联 =====

// ListMenuIDsByRoleID 查询某角色关联的 menu_id 列表。
func (ms *MenuStore) ListMenuIDsByRoleID(roleID int64) ([]int64, error) {
	var ids []int64
	err := ms.s.Query(
		`SELECT menu_id FROM role_menus WHERE role_id = :role_id`,
		map[string]any{"role_id": roleID},
	).Select(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// AssignMenu 给角色分配菜单。
func (ms *MenuStore) AssignMenu(roleID, menuID int64) error {
	_, err := ms.s.Query(
		`INSERT OR IGNORE INTO role_menus (role_id, menu_id) VALUES (:role_id, :menu_id)`,
		map[string]any{"role_id": roleID, "menu_id": menuID},
	).Exec()
	return err
}

// RevokeMenu 取消角色的菜单。
func (ms *MenuStore) RevokeMenu(roleID, menuID int64) error {
	_, err := ms.s.Query(
		`DELETE FROM role_menus WHERE role_id = :role_id AND menu_id = :menu_id`,
		map[string]any{"role_id": roleID, "menu_id": menuID},
	).Exec()
	return err
}

// DeleteRoleMenus 删除角色的全部菜单关联。
func (ms *MenuStore) DeleteRoleMenus(roleID int64) error {
	_, err := ms.s.Query(
		`DELETE FROM role_menus WHERE role_id = :role_id`,
		map[string]any{"role_id": roleID},
	).Exec()
	return err
}

// ===== 通用工具 =====

// inList 生成 SQL IN 占位符列表和对应参数 map。
// 如 inList("rid", []int64{1,2,3}) → (":rid_0, :rid_1, :rid_2", {"rid_0":1, "rid_1":2, "rid_2":3})
func inList(prefix string, ids []int64) (string, map[string]any) {
	placeholders := make([]string, len(ids))
	args := make(map[string]any, len(ids))
	for i, id := range ids {
		key := fmt.Sprintf("%s_%d", prefix, i)
		placeholders[i] = ":" + key
		args[key] = id
	}
	return strings.Join(placeholders, ", "), args
}
