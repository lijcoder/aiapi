package handler

import (
	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
)

// ParseUsage 从 Forward 缓存的完整原始响应中提取统一用量。
// 解析失败不影响已经完成的客户端响应，因此只跳过计费/用量记录。
func ParseUsage(ctx *types.Context) {
	if ctx.P == nil || !ctx.ResponseComplete || ctx.HttpResp == nil || ctx.HttpResp.StatusCode >= 300 {
		return
	}
	var usageErr error
	if ctx.Stream {
		ctx.Usage, usageErr = parser.ParseStreamUsage(ctx.P, ctx.RespBody)
	} else {
		ctx.Usage, usageErr = ctx.P.ParseUsage(ctx.RespBody)
	}
	if usageErr != nil {
		ctx.Usage = nil
	}
}
