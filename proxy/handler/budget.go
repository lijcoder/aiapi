package handler

import (
	"fmt"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

// BudgetCheck 请求前校验用户和 API Key 的余额
func BudgetCheck(ctx *types.Context) {
	key, user, err := service.NewApiKeyService().GetKeyAndUser(ctx.ApiKey)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if key == nil || user == nil {
		ctx.Err = log.WithStack(fmt.Errorf("invalid api key/user"))
		ctx.ErrorMessage = "invalid api key/user"
		ctx.Code = types.CodeUnauthorized
		return
	}

	ctx.UserUnlimited = user.Unlimited
	ctx.KeyUnlimited = key.Unlimited

	// 用户无限制 → 跳过所有校验
	if user.Unlimited {
		return
	}

	// 用户余额 ≤ 0 → 拒绝
	if user.Budget <= 0 {
		ctx.Err = log.WithStack(fmt.Errorf("insufficient user balance: user=%d budget=%.4f", user.ID, user.Budget))
		ctx.ErrorMessage = "insufficient user balance"
		ctx.Code = types.CodeInsufficientBalance
		return
	}

	// Key 有限额且花完了 → 拒绝
	if !key.Unlimited && key.Budget <= 0 {
		ctx.Err = log.WithStack(fmt.Errorf("insufficient key balance: key=%s budget=%.4f", ctx.ApiKey, key.Budget))
		ctx.ErrorMessage = "insufficient api key balance"
		ctx.Code = types.CodeInsufficientBalance
		return
	}
}
