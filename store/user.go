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
