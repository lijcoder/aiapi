package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// Charge 返回充值/扣费相关操作的命名空间。
func (s *Session) Charge() *ChargeStore {
	return &ChargeStore{s: s}
}

// ChargeStore 是充值/扣费相关操作的命名空间。
type ChargeStore struct {
	s *Session
}

// DeductUserBudget 扣减用户余额（amount 为正）
func (cs *ChargeStore) DeductUserBudget(userID int64, amount float64) error {
	_, err := cs.s.Query(
		`UPDATE users SET budget = budget - :amount WHERE id = :user_id`,
		map[string]any{"user_id": userID, "amount": amount},
	).Exec()
	return err
}

// RechargeUserBudget 充值用户余额（amount 为正）
func (cs *ChargeStore) RechargeUserBudget(userID int64, amount float64) error {
	_, err := cs.s.Query(
		`UPDATE users SET budget = budget + :amount WHERE id = :user_id`,
		map[string]any{"user_id": userID, "amount": amount},
	).Exec()
	return err
}

// DeductKeyBudget 扣减 API Key 余额（amount 为正）
func (cs *ChargeStore) DeductKeyBudget(key string, amount float64) error {
	_, err := cs.s.Query(
		`UPDATE api_keys SET budget = budget - :amount WHERE key = :key`,
		map[string]any{"key": key, "amount": amount},
	).Exec()
	return err
}

// InsertRechargeRecord 插入充值记录
func (cs *ChargeStore) InsertRechargeRecord(rec *model.RechargeRecord) error {
	res, err := cs.s.Query(
		`INSERT INTO recharge_records (user_id, amount, balance_before, balance_after, operator, remark)
		 VALUES (:user_id, :amount, :balance_before, :balance_after, :operator, :remark)`,
		rec,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	rec.ID = id
	return nil
}
