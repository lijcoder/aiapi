package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// 请求/响应结构体
type RechargeSelfReq struct {
	Amount float64 `json:"amount"`
	Remark string  `json:"remark"`
}

type RechargeAdminReq struct {
	UserID int64   `json:"userId"`
	Amount float64 `json:"amount"`
	Remark string  `json:"remark"`
}

type RecordsReq struct {
	UserID int64 `json:"userId"`
}

type RechargeResult struct {
	Record *model.RechargeRecord `json:"record"`
	Budget float64               `json:"budget"`
}

// RechargeSelf 普通用户给自己充值
func RechargeSelf(ctx context.Context, req *RechargeSelfReq) (*RechargeResult, *base.BizError) {
	cur := base.CurrentUser(ctx)
	result := &RechargeResult{}
	if req.Amount <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "amount must be positive")
	}
	err := store.T(func(s *store.Session) error {
		rec, err := s.Charge().RechargeWithRecord(cur.ID, req.Amount, cur.Account, req.Remark)
		if err != nil {
			return err
		}
		result.Record = rec
		return nil
	})
	if err != nil {
		slog.Error("recharge self failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	result.Budget = result.Record.BalanceAfter
	return result, nil
}

// RechargeAdmin 管理员给指定用户充值
func RechargeAdmin(ctx context.Context, req *RechargeAdminReq) (*RechargeResult, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.UserID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "user_id is required")
	}
	if req.Amount <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "amount must be positive")
	}
	target, err := store.C().User().GetByID(req.UserID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if target == nil {
		return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
	}
	result := &RechargeResult{}
	err = store.T(func(s *store.Session) error {
		rec, err := s.Charge().RechargeWithRecord(req.UserID, req.Amount, cur.Account, req.Remark)
		if err != nil {
			return err
		}
		result.Record = rec
		return nil
	})
	if err != nil {
		slog.Error("recharge admin failed", "err", err, "target_user_id", req.UserID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	result.Budget = result.Record.BalanceAfter
	return result, nil
}

// RechargeSelfRecords 查询自己的充值流水
func RechargeSelfRecords(ctx context.Context) ([]model.RechargeRecord, *base.BizError) {
	cur := base.CurrentUser(ctx)
	recs, err := store.C().Charge().ListRechargeRecords(cur.ID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return recs, nil
}

// RechargeRecords 管理员查询指定用户的充值流水
func RechargeRecords(ctx context.Context, req *RecordsReq) ([]model.RechargeRecord, *base.BizError) {
	recs, err := store.C().Charge().ListRechargeRecords(req.UserID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return recs, nil
}

// ListRechargeReq 全平台充值流水查询请求
type ListRechargeReq struct {
	Keyword string `json:"keyword"` // 按用户名/账号/备注模糊搜索
}

// ListRechargeRecords 超管查询全平台充值流水
func ListRechargeRecords(ctx context.Context, req *ListRechargeReq) ([]model.RechargeRecord, *base.BizError) {
	recs, err := store.C().Charge().ListAllRechargeRecords(strings.TrimSpace(req.Keyword))
	if err != nil {
		slog.Error("[Recharge] ListAll failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return recs, nil
}
