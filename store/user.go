package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetUserByID 按 ID 查询启用中的用户
func (s *Session) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	err := s.namedGet(&u, `SELECT * FROM users WHERE id = :id AND enabled = 1`, map[string]any{"id": id})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
