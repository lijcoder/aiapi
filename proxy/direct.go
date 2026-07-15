package proxy

import (
	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/handler"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Handle 代理入口：构建管道并执行
func Handle(req types.ProxyRequest) error {
	ctx := types.NewContext(req, parser.GetParser(req.Format))

	NewPipeline().
		AddLast(handler.Auth).
		AddLast(handler.LoadConfig).
		AddLast(handler.Forward).
		AddLast(handler.Response).
		AddLast(handler.Record).
		AddFinally(handler.Log).
		Execute(ctx)

	return nil
}
