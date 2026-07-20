package handler

import (
	"context"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// userInfo 用户基本信息
type userInfo struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Account string  `json:"account"`
	Budget  float64 `json:"budget"`
}

// SelfResp /manager/self 响应
type SelfResp struct {
	User        userInfo               `json:"user"`
	Permissions []model.RolePermission `json:"permissions"`
}

// Self 返回当前登录用户的实时信息（从数据库查询） + 权限列表。
func Self(ctx context.Context) (*SelfResp, *base.BizError) {
	curID := base.CurrentUser(ctx).ID
	user, err := store.C().User().GetByID(curID)
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if user == nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &SelfResp{
		User: userInfo{
			ID:      user.ID,
			Name:    user.Name,
			Account: user.Account,
			Budget:  user.Budget,
		},
		Permissions: base.CurrentPermissions(ctx),
	}, nil
}
