package handler

import (
	"log/slog"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

// ListModels 处理「列出可用模型」元数据端点（GET v1/models）。
// 数据源为本地 models 表（与 AuthModel 的可用性判定同源），不透传上游：
// 返回当前 provider 下、且当前 API Key 有权访问的模型，由 parser 序列化为对应协议格式。
func ListModels(ctx *types.Context) {
	models, err := service.NewModelService().ListAvailableModels(ctx.ProviderType, ctx.ApiKeyID)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	items := make([]parser.ModelItem, 0, len(models))
	for _, m := range models {
		items = append(items, parser.ModelItem{
			ID:        m.Model,
			CreatedAt: m.CreatedAt,
			OwnedBy:   m.Provider,
		})
	}
	body, err := parser.FormatModelList(ctx.P, items)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.Writer.WriteStatusCode(200)
	if _, err := ctx.Writer.Write(body); err != nil {
		if isBrokenPipe(err) {
			slog.Warn("client disconnected", "path", ctx.Path)
			return
		}
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "response write failed"
		return
	}
}
