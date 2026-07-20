package handler

import (
	"context"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
)

// userInfo 用户基本信息
type userInfo struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Account string  `json:"account"`
	Budget  float64 `json:"budget"`
}

// Self 返回当前登录用户的实时信息（从数据库查询，不使用 context 缓存）。
func Self(ctx context.Context) (*userInfo, *base.BizError) {
	curID := base.CurrentUser(ctx).ID
	user, err := store.C().User().GetByID(curID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if user == nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &userInfo{
		ID:      user.ID,
		Name:    user.Name,
		Account: user.Account,
		Budget:  user.Budget,
	}, nil
}
