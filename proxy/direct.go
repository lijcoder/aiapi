package proxy

import (
	"log/slog"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/handler"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Handle 代理入口：构建管道并执行
func Handle(req types.ProxyRequest) error {
	ctx := types.NewContext(req, parser.GetParser(req.Format))

	NewPipeline().
		AddLast(handler.ParseRequest).
		AddLast(handler.Auth).
		AddLast(handler.LoadConfig).
		AddLast(handler.Forward).
		AddLast(handler.Response).
		AddLast(handler.Record).
		AddFinally(handler.Log).
		Execute(ctx)

	// 整个管道执行完毕，统一打印错误日志
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

	return nil
}
