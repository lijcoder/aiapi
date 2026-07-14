package proxy

import (
	"database/sql"
	"net/http"

	"github.com/lijcoder/aiapi/parser"
)

// ProxyResponseWrite 响应写入接口
type ProxyResponseWrite interface {
	Header() http.Header
	WriteStatusCode(statusCode int)
	Write(body []byte) (int, error)
}

// ProxyRequest 代理请求参数
type ProxyRequest struct {
	DB       *sql.DB             // 数据库连接
	Format   string              // API 协议: openai / gemini / anthropic
	Provider string              // 后端 provider type
	Method   string              // HTTP 方法
	Path     string              // 上游请求路径
	Body     []byte              // 请求体
	Query    map[string][]string // 查询参数
	Writer   ProxyResponseWrite  // 响应写入器
	ApiKey   string              // 调用来源 API Key
}

// Handle 代理主入口，使用链式调用完成代理全流程
func Handle(req ProxyRequest) error {
	p := parser.GetParser(req.Format)

	chain := NewChain(req.DB, p, req.Writer).
		WithRequest(req.Method, req.Path, req.Body, req.Query).
		WithAuth(req.ApiKey)

	chain.
		LoadConfig(req.Provider).
		Forward()

	// 链中错误统一处理
	if chain.Error() != nil {
		chain.WriteError()
		return nil
	}

	if chain.IsStreaming() {
		chain.stream = true
		chain.StreamResponse()
		if chain.StatusCode() < 300 {
			chain.Record()
		}
	} else {
		chain.stream = false
		chain.Intercept()
		if chain.StatusCode() < 300 {
			chain.Record()
		}
		chain.WriteResponse()
	}

	return chain.Error()
}
