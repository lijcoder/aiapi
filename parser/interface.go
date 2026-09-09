package parser

import (
	"strings"
	"time"
)

// API 协议格式常量
const (
	FormatOpenAI    = "openai"
	FormatGemini    = "gemini"
	FormatAnthropic = "anthropic"
	// FormatResponses OpenAI Responses API 协议格式
	FormatResponses = "openai-responses"
)

// Usage 统一的用量结构
type Usage struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	TotalTokens     int    `json:"total_tokens"`
	RequestID       string `json:"request_id"`
	CachedTokens    int    `json:"cached_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens"`
}

// StreamEvent 流式 SSE 事件
type StreamEvent struct {
	EventType string // content / usage / done / error
	Content   string // delta 文本
	Usage     *Usage // 仅在最后一条事件中有
	Model     string // 模型名
}

// Parser 各 Provider 的请求/响应解析器接口
type Parser interface {
	// ParseModel 从请求 body 中提取模型名
	ParseModel(body []byte) string

	// ParseApiKey 从请求头中提取 API Key
	ParseApiKey(headers map[string][]string) string

	// ParseUsage 从非流式响应 body 中提取用量
	ParseUsage(body []byte) (*Usage, error)

	// ParseStreamEvent 解析单条 SSE data 行
	// 返回 nil, nil 表示当前行不包含需要处理的事件
	ParseStreamEvent(data []byte) (*StreamEvent, error)
}

// ModelItem 协议中立的模型条目，用于「列出模型」端点的响应序列化。
// 由业务层从 store 模型映射而来，parser 不反向依赖 store。
type ModelItem struct {
	ID        string    // 模型名
	CreatedAt time.Time // 创建时间
	OwnedBy   string    // 所属方（provider 标识）
}

// ModelsFormatter 可选接口：支持「列出模型」端点响应序列化的 Parser 实现它。
// 各协议响应形状不同（OpenAI 为 {object:"list",data:[...]}），通过类型断言使用，
// 不影响未实现的 Parser（如 Gemini TODO）。
type ModelsFormatter interface {
	FormatModels(items []ModelItem) ([]byte, error)
}

// FormatModelList 序列化模型列表响应：优先用给定 Parser 的 ModelsFormatter 实现，
// 未实现时回退 OpenAI 格式（调用方最常见的协议形态）。
func FormatModelList(p Parser, items []ModelItem) ([]byte, error) {
	if f, ok := p.(ModelsFormatter); ok {
		return f.FormatModels(items)
	}
	return OpenAI.(ModelsFormatter).FormatModels(items)
}

// 无状态 Parser，包级单例
var (
	OpenAI    Parser = &OpenAIParser{}
	Anthropic Parser = &AnthropicParser{}
	Responses Parser = &ResponsesParser{}
	// Gemini  Parser = &GeminiParser{}     // TODO
)

// GetParser 根据 API 协议格式获取对应的 Parser
func GetParser(format string) Parser {
	switch format {
	case FormatOpenAI:
		return OpenAI
	case FormatGemini:
		return nil // TODO
	case FormatAnthropic:
		return Anthropic
	case FormatResponses:
		return Responses
	default:
		return nil
	}
}

// headerGet 从 map[string][]string 头部中获取指定 key 的值（不区分大小写）
func headerGet(headers map[string][]string, key string) string {
	lower := strings.ToLower(key)
	for k, vs := range headers {
		if strings.ToLower(k) == lower && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// extractBearerToken 从 Authorization 头中提取 Bearer token
func extractBearerToken(headers map[string][]string) string {
	v := headerGet(headers, "Authorization")
	if len(v) > 7 && strings.ToLower(v[:7]) == "bearer " {
		return v[7:]
	}
	return v
}
