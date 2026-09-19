package handler

import (
	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

// RewriteModel 把请求体中的模型名改写为上游模型名（models.provider_model）。
//
// 对用户可见的 model 用于鉴权、计价、白名单与用量/日志口径，这里只改发往上游的
// 请求体，不改 ctx.Model；未配置 provider_model 时目标名等于 model，直接跳过
// （绝大多数请求零开销、请求体字节零变动）。
func RewriteModel(ctx *types.Context) {
	if ctx.P == nil || ctx.ModelInfo == nil {
		return
	}
	target := service.UpstreamModelName(ctx.ModelInfo)
	if target == ctx.Model {
		return
	}
	body, err := ctx.P.ReplaceModel(ctx.Body, target)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "failed to apply provider model"
		ctx.Code = types.CodeUnknown
		return
	}
	ctx.Body = body
}
