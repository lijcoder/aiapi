package proxy

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/parser"
)

// Chain 代理处理链，每个步骤返回 *Chain 支持链式调用
// 任一步骤出错，后续步骤跳过
type Chain struct {
	// 输入
	db     *sql.DB
	p      parser.Parser
	writer ProxyResponseWrite
	userID int64
	apiKey string

	method string
	path   string
	body   []byte
	query  map[string][]string

	// 步骤间状态
	providerType string
	url          string
	reqHeaders   map[string][]string
	httpResp     *http.Response
	respBody     []byte
	usage        *parser.Usage
	stream       bool

	err error
}

// NewChain 创建代理链
func NewChain(database *sql.DB, p parser.Parser, writer ProxyResponseWrite) *Chain {
	return &Chain{
		db:     database,
		p:      p,
		writer: writer,
	}
}

// WithRequest 设置请求信息
func (c *Chain) WithRequest(method, path string, body []byte, query map[string][]string) *Chain {
	c.method = method
	c.path = path
	c.body = body
	c.query = query
	return c
}

// WithAuth 设置认证信息（API Key）
func (c *Chain) WithAuth(apiKey string) *Chain {
	c.apiKey = apiKey
	c.userID = 0 // TODO: 内部查询后设置
	return c
}

// LoadConfig 从数据库加载 Provider 配置
func (c *Chain) LoadConfig(providerType string) *Chain {
	if c.err != nil {
		return c
	}
	c.providerType = providerType

	pvd, ok := db.GetProvider(c.db, providerType)
	if !ok {
		c.err = fmt.Errorf("provider not found: %s", providerType)
		return c
	}

	cfg, err := pvd.ParseConfig()
	if err != nil {
		c.err = fmt.Errorf("invalid provider config: %s", err.Error())
		return c
	}
	c.url = cfg.Domain + "/" + c.path
	c.reqHeaders = cfg.Headers
	return c
}

// Forward 转发请求到上游
func (c *Chain) Forward() *Chain {
	if c.err != nil {
		return c
	}

	req, err := http.NewRequest(c.method, c.url, io.NopCloser(bytes.NewReader(c.body)))
	if err != nil {
		c.err = err
		return c
	}

	req.Header = make(http.Header)
	for k, vs := range c.reqHeaders {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	q := req.URL.Query()
	for k, vs := range c.query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.err = err
		return c
	}
	c.httpResp = resp
	return c
}

// Intercept 拦截响应并解析 Token 用量（非流式）
func (c *Chain) Intercept() *Chain {
	if c.err != nil {
		return c
	}
	if c.p == nil {
		return c
	}

	body, err := io.ReadAll(c.httpResp.Body)
	if err != nil {
		c.err = err
		return c
	}
	c.httpResp.Body.Close()
	c.respBody = body

	usage, err := c.p.ParseUsage(body)
	if err != nil {
		return c
	}
	if usage != nil {
		c.usage = usage
	}
	return c
}

// Record 异步记录 Token 用量
func (c *Chain) Record() *Chain {
	if c.usage == nil {
		return c
	}
	db.RecordUsageAsync(c.db, &db.UsageRecord{
		UserID:       c.userID,
		ApiKey:       c.apiKey,
		Provider:     c.providerType,
		Model:        c.usage.Model,
		InputTokens:  c.usage.InputTokens,
		OutputTokens: c.usage.OutputTokens,
		TotalTokens:  c.usage.TotalTokens,
		RequestID:    c.usage.RequestID,
		Stream:       c.stream,
	})
	return c
}

// WriteResponse 将响应写回客户端
func (c *Chain) WriteResponse() *Chain {
	if c.err != nil {
		return c
	}
	for k, vs := range c.httpResp.Header {
		for _, v := range vs {
			c.writer.Header().Add(k, v)
		}
	}
	c.writer.WriteStatusCode(c.httpResp.StatusCode)

	if len(c.respBody) > 0 {
		_, c.err = c.writer.Write(c.respBody)
	}
	return c
}

// StreamResponse 流式处理：包装 sseBody，边读边写边解析
func (c *Chain) StreamResponse() *Chain {
	if c.err != nil {
		return c
	}

	for k, vs := range c.httpResp.Header {
		for _, v := range vs {
			c.writer.Header().Add(k, v)
		}
	}
	c.writer.WriteStatusCode(c.httpResp.StatusCode)

	var body io.ReadCloser = c.httpResp.Body
	if c.p != nil {
		body = newSSEBody(c.httpResp.Body, c.p)
	}

	buf := make([]byte, 512)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := c.writer.Write(buf[:n]); writeErr != nil {
				c.err = writeErr
				return c
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			c.err = err
			return c
		}
	}
	body.Close()

	if sb, ok := body.(*sseBody); ok {
		if u := sb.Usage(); u != nil {
			c.usage = u
		}
	}
	return c
}

// IsStreaming 判断响应是否流式
func (c *Chain) IsStreaming() bool {
	if c.httpResp == nil {
		return false
	}
	return strings.Contains(c.httpResp.Header.Get("Content-Type"), "event-stream")
}

// StatusCode 返回响应状态码
func (c *Chain) StatusCode() int {
	if c.httpResp == nil {
		return 0
	}
	return c.httpResp.StatusCode
}

// Error 返回链中累积的错误
func (c *Chain) Error() error {
	return c.err
}

// WriteError 将链中的错误转为 HTTP 响应写给客户端
func (c *Chain) WriteError() {
	if c.err == nil {
		return
	}
	status := http.StatusInternalServerError
	msg := c.err.Error()
	if len(msg) > 13 && msg[:13] == "provider not found" {
		status = http.StatusNotFound
	}

	body := fmt.Sprintf(`{"error":{"message":"%s"}}`, msg)
	c.writer.Header().Set("Content-Type", "application/json")
	c.writer.WriteStatusCode(status)
	c.writer.Write([]byte(body))
}
