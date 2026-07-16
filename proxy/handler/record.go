package handler

import (
	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// Record 记录 Token 用量、计算费用、扣减余额
// 注意：Record 在响应之后执行，所有错误收集到 OtherErrs
func Record(ctx *types.Context) {
	if ctx.Usage == nil || ctx.ModelInfo == nil {
		return
	}
	if ctx.HttpResp != nil && ctx.HttpResp.StatusCode >= 300 {
		return
	}

	// 1. 计算花费
	inputMiss := ctx.Usage.InputTokens - ctx.Usage.CachedTokens
	cost := (float64(ctx.Usage.CachedTokens)*ctx.ModelInfo.InputCacheHitPrice +
		float64(inputMiss)*ctx.ModelInfo.InputCacheMissPrice +
		float64(ctx.Usage.OutputTokens)*ctx.ModelInfo.OutputPrice) / 1_000_000

	// 2. 记录用量（始终写入）
	rec := &store.UsageRecord{
		UserID:          ctx.UserID,
		ApiKey:          ctx.ApiKey,
		Provider:        ctx.ProviderType,
		Model:           ctx.Model,
		InputTokens:     ctx.Usage.InputTokens,
		OutputTokens:    ctx.Usage.OutputTokens,
		TotalTokens:     ctx.Usage.TotalTokens,
		RequestID:       ctx.Usage.RequestID,
		Stream:          ctx.Stream,
		CachedTokens:    ctx.Usage.CachedTokens,
		ReasoningTokens: ctx.Usage.ReasoningTokens,
		Cost:            cost,
		Unlimited:       ctx.UserUnlimited,
	}
	if err := store.InsertUsage(rec); err != nil {
		ctx.OtherErrs = append(ctx.OtherErrs, log.WithStack(err))
	}

	// 3. 用户无限制 → 不扣费
	if ctx.UserUnlimited {
		return
	}

	// 4. 扣用户余额
	if err := store.DeductUserBudget(ctx.UserID, cost); err != nil {
		ctx.OtherErrs = append(ctx.OtherErrs, log.WithStack(err))
	}

	// 5. Key 有限额 → 额外扣 Key
	if !ctx.KeyUnlimited {
		if err := store.DeductKeyBudget(ctx.ApiKey, cost); err != nil {
			ctx.OtherErrs = append(ctx.OtherErrs, log.WithStack(err))
		}
	}
}
