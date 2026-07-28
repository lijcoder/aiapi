package proxy

import (
	"log/slog"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/handler"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Handle 转发请求入口：转发到上游并透传响应（由框架层通配路由进入）
func Handle(req types.ProxyRequest) error {
	ctx := types.NewContext(req, parser.GetParser(req.Format))

	NewPipeline().
		AddLast(handler.ParseRequest).
		AddLast(handler.AuthKey).
		AddLast(handler.AuthModel).
		AddLast(handler.BudgetCheck).
		AddLast(handler.LoadConfig).
		AddLast(handler.Forward).
		AddLast(handler.Response).
		AddLast(handler.Record).
		AddFinally(handler.Log).
		Execute(ctx)

	logErrors(ctx)
	return nil
}

// HandleModels 模型列表端点入口（GET v1/models，由框架层具体路由进入）：
// 返回本地 models 表中当前 Key 可用的模型，不经上游转发、不计费、不写请求日志
func HandleModels(req types.ProxyRequest) error {
	ctx := types.NewContext(req, parser.GetParser(req.Format))

	NewPipeline().
		AddLast(handler.ParseRequest).
		AddLast(handler.AuthKey).
		AddLast(handler.ListModels).
		Execute(ctx)

	logErrors(ctx)
	return nil
}

// logErrors 管道执行完毕后统一打印错误日志
func logErrors(ctx *types.Context) {
	if ctx.Err != nil {
		slog.Error("proxy request failed",
			"provider", ctx.ProviderType,
			"path", ctx.Path,
			"status", ctx.Code.HTTPStatus(),
			"err", ctx.Err.Error(),
		)
	}
	for _, oerr := range ctx.OtherErrs {
		slog.Error("proxy request error", "err", oerr.Error())
	}
}
