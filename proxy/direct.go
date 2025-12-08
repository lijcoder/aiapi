package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

type ProxyDirectRequest struct {
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

// 1、type 获取模型配置
// 2、构建 url, domain + uri
// 3、创建 request
// 4、request 添加 Headers
// 5、request 添加 query params
// 6、http 请求
// 7、获取响应。如何处理流式消息？
// 7.1、response Headers
// 7.2、response body. 流式如何处理？
func Direct(pdr ProxyDirectRequest) (*ProxyDirectResponse, error) {
	// 通过 type 获取模型配置
	modelConfig, flag := getModelConfig(pdr.Type)
	if !flag {
		return nil, errors.New("model config not found. type: " + pdr.Type)
	}
	domain := modelConfig.Domain
	url := domain + "/" + pdr.Path
	bodyReader := io.NopCloser(bytes.NewReader(pdr.Body))
	req, error := http.NewRequest(pdr.Method, url, bodyReader)
	if error != nil {
		return nil, error
	}
	// 添加查询参数
	query := req.URL.Query()
	for k, vs := range pdr.QueryParams {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	req.URL.RawQuery = query.Encode()
	req.Header = modelConfig.Headers
	client := &http.Client{}
	resp, error := client.Do(req)
	if error != nil {
		return nil, error
	}
	pdresp := ProxyDirectResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
	}
	return &pdresp, nil
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
