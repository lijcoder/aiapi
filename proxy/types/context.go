package types

import (
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
}

// Context 管道上下文，承载所有请求/响应状态
type Context struct {
	P      parser.Parser
	Writer ProxyResponseWrite

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
	return &Context{
		P:            p,
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
