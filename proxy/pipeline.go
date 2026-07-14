package proxy

import (
	"fmt"

	"github.com/lijcoder/aiapi/proxy/types"
)

// HandlerFunc 管道处理函数
type HandlerFunc func(ctx *types.Context)

// Pipeline 有序管道
type Pipeline struct {
	handlers []HandlerFunc
	finally  []HandlerFunc
}

// NewPipeline 创建空管道
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// AddLast 追加 handler 到管道末尾
func (p *Pipeline) AddLast(h HandlerFunc) *Pipeline {
	p.handlers = append(p.handlers, h)
	return p
}

// AddFinally 追加始终执行的收尾 handler（不受中断影响）
func (p *Pipeline) AddFinally(h HandlerFunc) *Pipeline {
	p.finally = append(p.finally, h)
	return p
}

// Execute 遍历执行所有 handler，遇错误终止
func (p *Pipeline) Execute(ctx *types.Context) {
	for _, h := range p.handlers {
		h(ctx)
		if ctx.Err != nil {
			break
		}
	}

	p.writeError(ctx)
	for _, f := range p.finally {
		f(ctx)
	}
}

// writeError 将错误转为 HTTP 响应
func (p *Pipeline) writeError(ctx *types.Context) {
	if ctx.Err == nil {
		return
	}
	body := fmt.Sprintf(`{"error":{"message":"%s"}}`, ctx.Err.Error())
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.Writer.WriteStatusCode(ctx.Code.HTTPStatus())
	ctx.Writer.Write([]byte(body))
}
