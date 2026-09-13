package handler

import (
	"time"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

// Record 记录 Token 用量、计算费用、扣减余额
// 注意：Record 在响应之后执行，所有错误收集到 OtherErrs
func Record(ctx *types.Context) {
	if ctx.Usage == nil || ctx.ModelInfo == nil || !ctx.ResponseComplete {
		return
	}
	if ctx.HttpResp != nil && ctx.HttpResp.StatusCode >= 300 {
		return
	}
	ctx.MarkLatency(time.Now())

	if _, err := service.NewBillingService().RecordUsage(service.UsageBillingInput{
		Model:            ctx.ModelInfo,
		UserID:           ctx.UserID,
		APIKeyID:         ctx.ApiKeyID,
		Provider:         ctx.ProviderType,
		ModelName:        ctx.Model,
		InputTokens:      ctx.Usage.InputTokens,
		OutputTokens:     ctx.Usage.OutputTokens,
		TotalTokens:      ctx.Usage.TotalTokens,
		RequestID:        ctx.Usage.RequestID,
		Stream:           ctx.Stream,
		CachedTokens:     ctx.Usage.CachedTokens,
		ReasoningTokens:  ctx.Usage.ReasoningTokens,
		UserUnlimited:    ctx.UserUnlimited,
		KeyUnlimited:     ctx.KeyUnlimited,
		FirstTokenMs:     ctx.FirstTokenMs,
		LatencyMs:        ctx.LatencyMs,
		RequestStartedAt: ctx.StartTime,
	}); err != nil {
		ctx.OtherErrs = append(ctx.OtherErrs, log.WithStack(err))
	}
}
