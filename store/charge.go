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

// RechargeWithRecord 在当前 Session（应在事务中）内完成一次充值：
// 读取充值前余额 → 增加余额 → 写入充值流水。amount 必须由调用方保证 > 0。
// operator 为发起方账号（自充值写用户本人账号，管理员充值写管理员账号）。
func (cs *ChargeStore) RechargeWithRecord(userID int64, amount float64, operator, remark string) (*model.RechargeRecord, error) {
	var u model.User
	if err := cs.s.Query(
		`SELECT * FROM users WHERE id = :user_id`,
		map[string]any{"user_id": userID},
	).Get(&u); err != nil {
		return nil, err
	}

	if err := cs.RechargeUserBudget(userID, amount); err != nil {
		return nil, err
	}

	rec := &model.RechargeRecord{
		UserID:        userID,
		Amount:        amount,
		BalanceBefore: u.Budget,
		BalanceAfter:  u.Budget + amount,
		Operator:      operator,
		Remark:        remark,
	}
	if err := cs.InsertRechargeRecord(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// ListRechargeRecords 查询某用户的充值流水（倒序）
func (cs *ChargeStore) ListRechargeRecords(userID int64) ([]model.RechargeRecord, error) {
	var recs []model.RechargeRecord
	err := cs.s.Query(
		`SELECT r.*, u.name AS operator_name FROM recharge_records r LEFT JOIN users u ON r.operator = u.account WHERE r.user_id = :user_id ORDER BY r.id DESC`,
		map[string]any{"user_id": userID},
	).Select(&recs)
	if err != nil {
		return nil, err
	}
	return recs, nil
}

// ListAllRechargeRecords 查询全部充值流水（倒序）
func (cs *ChargeStore) ListAllRechargeRecords() ([]model.RechargeRecord, error) {
	var recs []model.RechargeRecord
	err := cs.s.Query(
		`SELECT * FROM recharge_records ORDER BY id DESC`,
		nil,
	).Select(&recs)
	if err != nil {
		return nil, err
	}
	return recs, nil
}

