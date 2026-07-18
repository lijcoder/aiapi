package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetUserByID 按 ID 查询启用中的用户
func (s *Session) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	err := dbFrom(s.Ctx).Get(&u, `SELECT * FROM users WHERE id = ? AND enabled = 1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserBudget 更新用户余额
func (s *Session) UpdateUserBudget(userID int64, budget float64) error {
	_, err := dbFrom(s.Ctx).Exec(`UPDATE users SET budget = ? WHERE id = ?`, budget, userID)
	return err
}
