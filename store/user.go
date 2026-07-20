package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// User 返回用户相关操作的命名空间。
func (s *Session) User() *UserStore {
	return &UserStore{s: s}
}

// UserStore 是用户相关操作的命名空间。
type UserStore struct {
	s *Session
}

// GetByID 按 ID 查询启用中的用户
func (us *UserStore) GetByID(id int64) (*model.User, error) {
	var u model.User
	err := us.s.Query(
		`SELECT * FROM users WHERE id = :id AND enabled = 1`,
		map[string]any{"id": id},
	).Get(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByAccount 按账号查询用户（不过滤 enabled，调用方自行判断启用状态）
func (us *UserStore) GetByAccount(account string) (*model.User, error) {
	var u model.User
	err := us.s.Query(
		`SELECT * FROM users WHERE account = :account`,
		map[string]any{"account": account},
	).Get(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePassword 更新用户密码哈希
func (us *UserStore) UpdatePassword(userID int64, passwordHash string) error {
	_, err := us.s.Query(
		`UPDATE users SET password = :password, updated_at = datetime('now','localtime') WHERE id = :user_id`,
		map[string]any{"user_id": userID, "password": passwordHash},
	).Exec()
	return err
}

// SetEnabled 启用/禁用用户
func (us *UserStore) SetEnabled(userID int64, enabled bool) error {
	_, err := us.s.Query(
		`UPDATE users SET enabled = :enabled, updated_at = datetime('now','localtime') WHERE id = :user_id`,
		map[string]any{"user_id": userID, "enabled": enabled},
	).Exec()
	return err
}
