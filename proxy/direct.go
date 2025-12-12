package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	modelConfigFile string
	modelConfig     []ProxyDirectModelConfig
)

func init() {
	modelConfigFile = initModelConfigFilePath(".aiapi/model_direct.json")
	modelConfig = initModelConfig()
}

type ProxyDirectModelConfig struct {
	Type    string              `json:"type"`
	Domain  string              `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

type ProxyDirect struct {
	Request       *ProxyDirectRequest
	Response      ProxyDirectResponseWrite
	proxyResponse *ProxyDirectResponse
}

type ProxyDirectRequest struct {
	Debug       bool
	TraceId     string
	Url         *url.URL
	Type        string
	Path        string
	Method      string
	Headers     http.Header
	QueryParams map[string][]string
	Body        []byte
}

type ProxyDirectResponse struct {
	Status     string
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
}

type ProxyDirectResponseWrite interface {
	Header() http.Header
	WriteStatusCode(statusCode int)
	Write(body []byte) (int, error)
}

// 1、type 获取模型配置
// 2、构建 url, domain + uri
// 3、创建 request
// 4、request 添加 Headers
// 5、request 添加 query params
// 6、http 请求
// 7、获取响应。如何处理流式消息？
// 7.1、response Headers
// 7.2、response body. 流式如何处理？
func (p *ProxyDirect) Direct() error {
	p.proxyTraceLog("RequestURL", p.Request.Url)
	p.proxyTraceLog("RequestPath", p.Request.Path)
	p.proxyTraceLog("RequestMethod", p.Request.Method)
	p.proxyTraceLog("RequestHeaders", p.Request.Headers)
	p.proxyTraceLog("RequestQueryParams", p.Request.QueryParams)
	p.proxyTraceLog("RequestBody", p.Request.Body)
	// 通过 type 获取模型配置
	modelConfig, flag := getModelConfig(p.Request.Type)
	if !flag {
		return errors.New("model config not found. type: " + p.Request.Type)
	}
	domain := modelConfig.Domain
	url := domain + "/" + p.Request.Path
	bodyReader := io.NopCloser(bytes.NewReader(p.Request.Body))
	req, error := http.NewRequest(p.Request.Method, url, bodyReader)
	if error != nil {
		return error
	}
	// 添加查询参数
	query := req.URL.Query()
	for k, vs := range p.Request.QueryParams {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	req.URL.RawQuery = query.Encode()
	req.Header = modelConfig.Headers
	client := &http.Client{}
	resp, error := client.Do(req)
	if error != nil {
		return error
	}
	defer resp.Body.Close()
	pdrs := ProxyDirectResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
	}
	p.proxyTraceLog("ResponseStatusCode", pdrs.StatusCode)
	p.proxyTraceLog("ResponseHeaders", pdrs.Headers)
	p.proxyResponse = &pdrs
	p.proxyResponseProcess()
	p.proxyTraceLog("RequestEnd", "------")
	return nil
}

func (p *ProxyDirect) proxyResponseProcess() error {
	// 设置 headers
	for k, vs := range p.proxyResponse.Headers {
		for _, v := range vs {
			p.Response.Header().Add(k, v)
		}
	}
	// 设置状态码，写响应头
	p.Response.WriteStatusCode(p.proxyResponse.StatusCode)
	// 根据 contentType 设置相应的响应格式
	contentType := p.proxyResponse.Headers.Get("Content-Type")
	sseContentType := "event-stream"
	if strings.Contains(contentType, sseContentType) {
		// stream
		return p.proxyResponseStream()
	} else {
		// no stream
		return p.proxyResponseNoStream()
	}
}

func (p *ProxyDirect) proxyResponseNoStream() error {
	bodyBytes, err := io.ReadAll(p.proxyResponse.Body)
	if err != nil {
		return err
	}
	p.proxyTraceLog("ResponseBody", string(bodyBytes))
	_, err = p.Response.Write(bodyBytes)
	return err
}

func (p *ProxyDirect) proxyResponseStream() error {
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	scratchBuf := make([]byte, 512)
	sep := []byte("\n\n")
	for {
		n, err := p.proxyResponse.Body.Read(scratchBuf)
		if n > 0 {
			buffer.Write(scratchBuf[:n])
			data := buffer.Bytes()
			for {
				idx := bytes.Index(data, sep)
				if idx == -1 {
					break
				}
				msgEnd := idx + 2
				msg := data[:msgEnd]
				p.proxyTraceLog("ResponseSSEBody", msg)
				if _, writeErr := p.Response.Write(msg); writeErr != nil {
					return writeErr
				}
				// 移除以处理的部分
				data = data[msgEnd:]
			}
			buffer.Reset()
			buffer.Write(data)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	if buffer.Len() > 0 {
		remaining := buffer.Bytes()
		p.proxyTraceLog("ResponseSSEBody", remaining)
		if _, writeErr := p.Response.Write(remaining); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func (p *ProxyDirect) proxyTraceLog(title string, data any) {
	if !p.Request.Debug {
		return
	}
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		json, _ := json.Marshal(v)
		dataStr = string(json)
	}
	logData := fmt.Sprintf("traceId:%s, title[%s]\\\\%s", p.Request.TraceId, title, dataStr)
	slog.Info(logData)
}

func getModelConfig(modelType string) (ProxyDirectModelConfig, bool) {
	for _, config := range modelConfig {
		if config.Type == modelType {
			return config, true
		}
	}
	// 如果没有找到，返回空结构体和 false
	return ProxyDirectModelConfig{}, false
}

func initModelConfig() []ProxyDirectModelConfig {
	// 读取文件内容
	content, err := os.ReadFile(modelConfigFile)
	if err != nil {
		// 如果文件不存在，直接终止程序
		panic("配置文件不存在: " + modelConfigFile + " 错误: " + err.Error())
	}
	// 解析JSON
	var configs []ProxyDirectModelConfig
	if err := json.Unmarshal(content, &configs); err != nil {
		// 解析失败时直接终止程序
		panic("配置文件解析失败: " + modelConfigFile + " 错误: " + err.Error())
	}
	return configs
}

func initModelConfigFilePath(modelConfigFile string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, modelConfigFile)
}
