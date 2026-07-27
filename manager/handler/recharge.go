package handler

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	base "github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// chargeService 与 middleware/handler 共用同一实例（无状态）。
var chargeService = service.NewChargeService()

type RechargeReq struct {
	UserID int64   `json:"userId"`
	Amount float64 `json:"amount"`
	Remark string  `json:"remark"`
}

type RecordsReq struct {
	UserID int64 `json:"userId"`
	base.PageReq
}

type RechargeResult struct {
	Record *model.RechargeRecord `json:"record"`
	Budget float64               `json:"budget"`
}

// RechargeSelf 普通用户给自己充值
func RechargeSelf(ctx context.Context, req *RechargeReq) (*RechargeResult, *base.BizError) {
	cur := base.CurrentUser(ctx)
	req.UserID = cur.ID
	return Recharge(ctx, req)
}

// Recharge 给指定用户充值
func Recharge(ctx context.Context, req *RechargeReq) (*RechargeResult, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.UserID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "user_id is required")
	}
	if req.Amount <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "amount must be positive")
	}
	rec, err := chargeService.RechargeWithRecord(req.UserID, req.Amount, cur.Account, req.Remark)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, base.NewBizError(base.CodeUserNotFound, "user not found")
		}
		slog.Error("recharge failed", "err", err, "target_user_id", req.UserID, "operator", cur.Account)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &RechargeResult{Record: rec, Budget: rec.BalanceAfter}, nil
}

// RechargeRecordsSelf 查询自己的充值流水（分页）
func RechargeRecordsSelf(ctx context.Context, req *RecordsReq) (*base.PageResult[model.RechargeRecord], *base.BizError) {
	cur := base.CurrentUser(ctx)
	req.UserID = cur.ID
	return RechargeRecords(ctx, req)
}

// RechargeRecords 查询指定用户的充值流水（分页）
func RechargeRecords(ctx context.Context, req *RecordsReq) (*base.PageResult[model.RechargeRecord], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	recs, err := store.C().SetPage(pc).Charge().ListRechargeRecords(req.UserID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &base.PageResult[model.RechargeRecord]{Items: recs, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// ListRechargeReq 全平台充值流水查询请求
type ListRechargeReq struct {
	Keyword string `json:"keyword"` // 按用户名/账号/备注模糊搜索
	base.PageReq
}

// ListRechargeRecords 超管查询全平台充值流水（分页）
func ListRechargeRecords(ctx context.Context, req *ListRechargeReq) (*base.PageResult[model.RechargeRecord], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	recs, err := store.C().SetPage(pc).Charge().ListAllRechargeRecords(strings.TrimSpace(req.Keyword))
	if err != nil {
		slog.Error("[Recharge] ListAll failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &base.PageResult[model.RechargeRecord]{Items: recs, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}
