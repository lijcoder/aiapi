// Package service 封装管理台通用业务逻辑，不依赖 echo，供 handler/middleware 调用。
//
// charge.go 实现充值/扣费相关业务：带流水的事务编排。
// store 层只提供单条 SQL（读余额、加余额、写流水），事务体在此组合。
package service

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ChargeService 封装充值/扣费业务逻辑。
// 无状态，进程内可共享单例。
type ChargeService struct{}

// NewChargeService 创建 ChargeService。
func NewChargeService() *ChargeService { return &ChargeService{} }

// RechargeWithRecord 完成一次充值：原子加余额 → 事务内读新余额 → 写流水。
//   - userID：被充值的用户
//   - amount：充值金额，必须 > 0（调用方校验）
//   - operator：发起方账号（自充值写用户本人账号，管理员充值写管理员账号）
//   - remark：备注
//
// 返回写入后的充值流水（含 ID 与 before/after 余额）。
//
// 并发安全原理（跨 SQLite/MySQL/PostgreSQL 通用）：
//   - 先 UPDATE 加余额（对同一行 UPDATE 加行锁串行执行，金额不丢更新）
//   - 再在同事务内 SELECT 新余额（事务内自己修改对后续语句立即可见）
//   - 行锁持续到 COMMIT，期间其他事务的 UPDATE 阻塞，故 SELECT 拿到的是本次更新后的稳定值
//   - before 由 after - amount 反推，不准读后写
func (s *ChargeService) RechargeWithRecord(userID int64, amount float64, operator, remark string) (*model.RechargeRecord, error) {
	var rec *model.RechargeRecord
	err := store.T(func(ss *store.Session) error {
		// 1. 原子加余额（行锁，串行化同一行的写）
		if err := ss.Charge().RechargeUserBudget(userID, amount); err != nil {
			return err
		}

		// 2. 事务内读新余额（自己事务的修改对后续语句可见，拿到的是 UPDATE 后的值）
		newBudget, err := ss.Charge().GetUserBudget(userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return err
		}

		// 3. 写流水，before 由 after 反推
		rec = &model.RechargeRecord{
			UserID:        userID,
			Amount:        amount,
			BalanceBefore: newBudget - amount,
			BalanceAfter:  newBudget,
			Operator:      operator,
			Remark:        remark,
		}
		return ss.Charge().InsertRechargeRecord(rec)
	})
	if err != nil {
		return nil, err
	}
	return rec, nil
}
