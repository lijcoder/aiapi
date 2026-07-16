package handler

import (
	"github.com/lijcoder/aiapi/proxy/types"
)

// ParseRequest 从请求中解析 API Key 和模型名
func ParseRequest(ctx *types.Context) {
	if ctx.P != nil {
		ctx.ApiKey = ctx.P.ParseApiKey(ctx.OrigHeaders)
		ctx.Model = ctx.P.ParseModel(ctx.Body)
	}
}
