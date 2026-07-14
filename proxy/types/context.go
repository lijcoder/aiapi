package types

import (
	"database/sql"
	"net/http"
	"time"

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
	DB        *sql.DB
	Format    string
	Provider  string
	Method    string
	Path      string
	Body      []byte
	Query     map[string][]string
	Writer    ProxyResponseWrite
	ApiKey    string
	Headers   map[string][]string
	StartTime time.Time
}

// Context 管道上下文，承载所有请求/响应状态
type Context struct {
	DB     *sql.DB
	P      parser.Parser
	Writer ProxyResponseWrite

	Method, Path, Format, ApiKey, ProviderType string
	Body                                       []byte
	Query                                      map[string][]string
	OrigHeaders                                map[string][]string
	StartTime                                  time.Time

	URL        string
	ReqHeaders map[string][]string
	HttpResp   *http.Response
	RespBody   []byte
	Usage      *parser.Usage
	Stream     bool

	Err  error
	Code BizCode
}

// NewContext 创建管道上下文
func NewContext(req ProxyRequest, p parser.Parser) *Context {
	return &Context{
		DB:           req.DB,
		P:            p,
		Writer:       req.Writer,
		Method:       req.Method,
		Path:         req.Path,
		Body:         req.Body,
		Query:        req.Query,
		Format:       req.Format,
		OrigHeaders:  req.Headers,
		StartTime:    req.StartTime,
		ApiKey:       req.ApiKey,
		ProviderType: req.Provider,
	}
}
