package types

import (
	"context"
	"net/http"
	"time"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/store/model"
)

// ProxyResponseWrite 响应写入接口
type ProxyResponseWrite interface {
	Header() http.Header
	WriteStatusCode(statusCode int)
	Write(body []byte) (int, error)
}

// ProxyRequest 代理请求参数
type ProxyRequest struct {
	Format    string
	Provider  string
	Method    string
	Path      string
	Body      []byte
	Query     map[string][]string
	Writer    ProxyResponseWrite
	Headers   map[string][]string
	StartTime time.Time

	// Ctx 请求生命周期 context（标准库类型，与具体 HTTP 框架无关）。
	// 由框架适配层注入（如 echo 适配层传 c.Request().Context()），
	// 客户端断开时框架会取消它，proxy 用它取消发往上游的请求。
	// 允许为 nil（如单元测试），NewContext 会兜底为 context.Background()。
	Ctx context.Context
}

// Context 管道上下文，承载所有请求/响应状态
type Context struct {
	P      parser.Parser
	Writer ProxyResponseWrite

	Ctx                                context.Context // 请求生命周期 context（见 ProxyRequest.Ctx）
	Method, Path, Format, ProviderType string
	ApiKey                             string
	Body                               []byte
	Query                              map[string][]string
	OrigHeaders                        map[string][]string
	StartTime                          time.Time
	UserID                             int64
	ApiKeyID                           int64
	UserUnlimited                      bool
	KeyUnlimited                       bool

	URL        string
	ReqHeaders map[string][]string
	Model      string       // 从请求体解析的模型名（始终有值）
	ModelInfo  *model.Model // 模型价格配置（Auth 校验后设置）
	HttpResp   *http.Response
	RespBody   []byte
	Usage      *parser.Usage // 从响应解析的用量（仅在成功请求时有值）
	Stream     bool

	Err          error   // 系统错误（管道中断 + 内部日志）
	Code         BizCode // HTTP 状态码
	ErrorMessage string  // 返回给客户端的错误描述，为空时使用固定提示
	OtherErrs    []error // 非中断性错误（finally 阶段收集，仅用于日志）
}

// NewContext 创建管道上下文
func NewContext(req ProxyRequest, p parser.Parser) *Context {
	c := req.Ctx
	if c == nil {
		c = context.Background()
	}
	return &Context{
		P:            p,
		Ctx:          c,
		Writer:       req.Writer,
		Method:       req.Method,
		Path:         req.Path,
		Body:         req.Body,
		Query:        req.Query,
		Format:       req.Format,
		OrigHeaders:  req.Headers,
		StartTime:    req.StartTime,
		ProviderType: req.Provider,
	}
}
