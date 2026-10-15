package service

import (
	"errors"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ===== 对外哨兵错误 =====
//
// handler 据此映射响应：
//   - ErrRoleNotFound → base.ErrBadReq("角色不存在")

// ErrRoleNotFound 表示 roleIDs 中存在 roles 表里不存在的角色。
var ErrRoleNotFound = errors.New("role not found")

// UserService 封装用户相关业务逻辑（事务编排）。
type UserService struct{}

// NewUserService 创建 UserService。
func NewUserService() *UserService { return &UserService{} }

// CreateUserWithRoles 创建用户并写入角色关联（一次写 users + user_roles 两张表，须在同一事务内）。
// roleIDs 为空时只建用户、不写角色关联。任一角色不存在返回 ErrRoleNotFound，
// 此时不落任何行、u.ID 保持 0。
//
// 角色存在性校验刻意放在事务**之外**（非事务只读），事务体内只有写操作：
//   - 生产 DSN 是 `_journal_mode=WAL&_busy_timeout=5000`，sqlx 的 Beginx 开的是 BEGIN DEFERRED。
//     事务内"先 SELECT 再 INSERT"时 SELECT 取到只读快照，INSERT 需升级写锁，
//     快照失效时 SQLite 直接返回 SQLITE_BUSY_SNAPSHOT —— 该错误不吃 `busy_timeout`，
//     表现为并发创建带 role_ids 的请求随机 500「系统繁忙」。
//     事务体只写，与本仓其它 service 事务（service/charge.go 的"先写后读"、docs/architecture.md §6）一致。
//   - 取舍：校验与写入之间存在窗口，但角色是种子数据且当前没有删除角色的接口，窗口内角色不会消失，
//     故校验通过后 AssignRole 必然有效。将来若新增"删除角色"接口，需把校验移入"先写后读"的事务，
//     或给 DSN 加 `_txlock=immediate`（后者改全局行为，当前不做）。
func (s *UserService) CreateUserWithRoles(u *model.User, roleIDs []int64) error {
	// 事务外校验，避免 WAL + deferred 事务下"先读后写"的写锁升级失败
	for _, rid := range roleIDs {
		r, err := store.C().Role().GetByID(rid)
		if err != nil {
			return err
		}
		if r == nil {
			return ErrRoleNotFound
		}
	}

	// 事务体内只写
	return store.C().T(func(ss *store.Session) error {
		if err := ss.User().Create(u); err != nil {
			return err
		}

		for _, rid := range roleIDs {
			if err := ss.Role().AssignRole(u.ID, rid); err != nil {
				return err
			}
		}
		return nil
	})
}
