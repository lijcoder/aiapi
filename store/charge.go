package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// DeductUserBudget 扣减用户余额（amount 为正）
func (s *Session) DeductUserBudget(userID int64, amount float64) error {
	_, err := s.namedExec(
		`UPDATE users SET budget = budget - :amount WHERE id = :user_id`,
		map[string]any{"user_id": userID, "amount": amount},
	)
	return err
}

// RechargeUserBudget 充值用户余额（amount 为正）
func (s *Session) RechargeUserBudget(userID int64, amount float64) error {
	_, err := s.namedExec(
		`UPDATE users SET budget = budget + :amount WHERE id = :user_id`,
		map[string]any{"user_id": userID, "amount": amount},
	)
	return err
}

// DeductKeyBudget 扣减 API Key 余额（amount 为正）
func (s *Session) DeductKeyBudget(key string, amount float64) error {
	_, err := s.namedExec(
		`UPDATE api_keys SET budget = budget - :amount WHERE key = :key`,
		map[string]any{"key": key, "amount": amount},
	)
	return err
}

// InsertRechargeRecord 插入充值记录
func (s *Session) InsertRechargeRecord(rec *model.RechargeRecord) error {
	_, err := s.namedExec(
		`INSERT INTO recharge_records (user_id, amount, balance_before, balance_after, operator, remark)
		 VALUES (:user_id, :amount, :balance_before, :balance_after, :operator, :remark)`,
		rec,
	)
	return err
}
