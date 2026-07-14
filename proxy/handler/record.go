package handler

import (
	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Record 异步记录 Token 用量
func Record(ctx *types.Context) {
	if ctx.Usage == nil {
		return
	}
	if ctx.HttpResp != nil && ctx.HttpResp.StatusCode >= 300 {
		return
	}
	db.RecordUsageAsync(ctx.DB, &db.UsageRecord{
		ApiKey:       ctx.ApiKey,
		Provider:     ctx.ProviderType,
		Model:        ctx.Usage.Model,
		InputTokens:  ctx.Usage.InputTokens,
		OutputTokens: ctx.Usage.OutputTokens,
		TotalTokens:  ctx.Usage.TotalTokens,
		RequestID:    ctx.Usage.RequestID,
		Stream:       ctx.Stream,
	})
}
