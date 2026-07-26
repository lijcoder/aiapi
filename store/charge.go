package store

import (
	"strings"

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

// DeductKeyBudget 扣减 API Key 余额（amount 为正）
func (cs *ChargeStore) DeductKeyBudget(key string, amount float64) error {
	_, err := cs.s.Query(
		`UPDATE api_keys SET budget = budget - :amount WHERE key = :key`,
		map[string]any{"key": key, "amount": amount},
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

// GetUserBudget 查询用户余额
func (cs *ChargeStore) GetUserBudget(userID int64) (float64, error) {
	var budget float64
	err := cs.s.Query(
		`SELECT budget FROM users WHERE id = :user_id`,
		map[string]any{"user_id": userID},
	).Get(&budget)
	return budget, err
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

// ListAllRechargeRecords 查询全部充值流水（倒序），支持按用户名/账号/备注模糊搜索
func (cs *ChargeStore) ListAllRechargeRecords(keyword string) ([]model.RechargeRecord, error) {
	q := `SELECT r.*, u.name AS operator_name, u2.name AS user_name
		 FROM recharge_records r
		 LEFT JOIN users u ON r.operator = u.account
		 LEFT JOIN users u2 ON r.user_id = u2.id`
	args := map[string]any{}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q += ` WHERE u2.name LIKE :kw OR u2.account LIKE :kw OR r.remark LIKE :kw`
		args["kw"] = "%" + kw + "%"
	}
	q += ` ORDER BY r.id DESC`
	var recs []model.RechargeRecord
	err := cs.s.Query(q, args).Select(&recs)
	if err != nil {
		return nil, err
	}
	return recs, nil
}
