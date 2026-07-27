package service

import "github.com/lijcoder/aiapi/store"

// MenuService 封装菜单相关业务逻辑（事务编排）。
type MenuService struct{}

// NewMenuService 创建 MenuService。
func NewMenuService() *MenuService { return &MenuService{} }

// ReplaceRoleMenus 全量替换角色的菜单关联：删旧 → 循环插新，事务内执行。
func (s *MenuService) ReplaceRoleMenus(roleID int64, menuIDs []int64) error {
	return store.C().T(func(ss *store.Session) error {
		if err := ss.Menu().DeleteRoleMenus(roleID); err != nil {
			return err
		}
		for _, mid := range menuIDs {
			if err := ss.Menu().AssignMenu(roleID, mid); err != nil {
				return err
			}
		}
		return nil
	})
}
