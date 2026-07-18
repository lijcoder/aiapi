package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// InsertRechargeRecord 插入充值记录
func (s *Session) InsertRechargeRecord(rec *model.RechargeRecord) error {
	_, err := dbFrom(s.Ctx).NamedExec(
		`INSERT INTO recharge_records (user_id, amount, balance_before, balance_after, operator, remark)
		 VALUES (:user_id, :amount, :balance_before, :balance_after, :operator, :remark)`,
		rec,
	)
	return err
}
