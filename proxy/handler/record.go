package handler

import (
	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// Record 记录 Token 用量
func Record(ctx *types.Context) {
	if ctx.Usage == nil {
		return
	}
	if ctx.HttpResp != nil && ctx.HttpResp.StatusCode >= 300 {
		return
	}
	if err := store.InsertUsage(&store.UsageRecord{
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
	}); err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
	}
}
