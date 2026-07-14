package proxy

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/parser"
)

// Chain 代理处理链，每个步骤返回 *Chain 支持链式调用
// 任一步骤出错，后续步骤跳过
type Chain struct {
	// 输入
	db      *sql.DB
	p       parser.Parser
	writer  ProxyResponseWrite
	userID  int64
	apiKey  string
	debug   bool
	traceId string

	method  string
	path    string
	body    []byte
	query   map[string][]string
	headers http.Header // 原始请求头（调试用）

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

// WithUser 设置用户 ID
func (c *Chain) WithUser(userID int64) *Chain {
	c.userID = userID
	return c
}

// WithApiKey 设置调用来源的 API Key
func (c *Chain) WithApiKey(apiKey string) *Chain {
	c.apiKey = apiKey
	return c
}

// WithDebug 开启调试日志
func (c *Chain) WithDebug(traceId string) *Chain {
	c.debug = true
	c.traceId = traceId
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
	c.traceLog("LoadConfig", map[string]string{"url": c.url, "headers": fmt.Sprintf("%v", c.reqHeaders)})
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

	// 注入 Provider 配置的 Headers
	req.Header = make(http.Header)
	for k, vs := range c.reqHeaders {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// 追加查询参数
	q := req.URL.Query()
	for k, vs := range c.query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()

	c.traceLog("Forward", req.URL.String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.err = err
		return c
	}
	c.httpResp = resp
	c.traceLog("Forward.Status", resp.Status)
	return c
}

// Intercept 拦截响应并解析 Token 用量（非流式）
func (c *Chain) Intercept() *Chain {
	if c.err != nil {
		return c
	}
	if c.p == nil {
		return c // 没有 parser，跳过解析
	}

	body, err := io.ReadAll(c.httpResp.Body)
	if err != nil {
		c.err = err
		return c
	}
	c.httpResp.Body.Close()
	c.respBody = body
	c.traceLog("Intercept.Body", string(body))

	usage, err := c.p.ParseUsage(body)
	if err != nil {
		c.traceLog("Intercept.ParseError", err.Error())
		return c // 解析失败不影响响应
	}
	if usage != nil {
		c.usage = usage
		c.traceLog("Intercept.Usage", usage)
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
	c.traceLog("Record", "async queued")
	return c
}

// WriteResponse 将响应写回客户端
func (c *Chain) WriteResponse() *Chain {
	if c.err != nil {
		return c
	}
	// 写 Headers
	for k, vs := range c.httpResp.Header {
		for _, v := range vs {
			c.writer.Header().Add(k, v)
		}
	}
	c.writer.WriteStatusCode(c.httpResp.StatusCode)

	// 非流式：写完整 body
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

	// 写 Headers
	for k, vs := range c.httpResp.Header {
		for _, v := range vs {
			c.writer.Header().Add(k, v)
		}
	}
	c.writer.WriteStatusCode(c.httpResp.StatusCode)

	// 创建 SSE 包装器
	var body io.ReadCloser = c.httpResp.Body
	if c.p != nil {
		body = newSSEBody(c.httpResp.Body, c.p)
	}

	// 逐块读取并写出
	buf := make([]byte, 512)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			c.traceLog("Stream.Chunk", string(buf[:n]))
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

	// 从 sseBody 提取用量
	if sb, ok := body.(*sseBody); ok {
		if u := sb.Usage(); u != nil {
			c.usage = u
		}
	}
	c.traceLog("Stream.Done", "stream ended")
	return c
}

// IsStreaming 判断响应是否流式
func (c *Chain) IsStreaming() bool {
	if c.httpResp == nil {
		return false
	}
	return strings.Contains(c.httpResp.Header.Get("Content-Type"), "event-stream")
}

// ContentType 返回响应的 Content-Type
func (c *Chain) ContentType() string {
	if c.httpResp == nil {
		return ""
	}
	return c.httpResp.Header.Get("Content-Type")
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
	// 根据错误类型映射状态码
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

// traceLog 调试日志
func (c *Chain) traceLog(title string, data any) {
	if !c.debug {
		return
	}
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		j, _ := json.Marshal(v)
		dataStr = string(j)
	}
	slog.Info(fmt.Sprintf("traceId:%s, step[%s]\\\\%s", c.traceId, title, dataStr))
}
