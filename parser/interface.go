package parser

import "net/http"

// API 协议格式常量
const (
	FormatOpenAI    = "openai"
	FormatGemini    = "gemini"
	FormatAnthropic = "anthropic"
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
	EventType string  // content / usage / done / error
	Content   string  // delta 文本
	Usage     *Usage  // 仅在最后一条事件中有
	Model     string  // 模型名
}

// Parser 各 Provider 的响应解析器接口
type Parser interface {
	// ParseUsage 从非流式响应 body 中提取用量
	ParseUsage(body []byte) (*Usage, error)

	// ParseStreamEvent 解析单条 SSE data 行
	// 返回 nil, nil 表示当前行不包含需要处理的事件
	ParseStreamEvent(data []byte) (*StreamEvent, error)
}

// 无状态 Parser，包级单例
var (
	OpenAI    Parser = &OpenAIParser{}
	Anthropic Parser = &AnthropicParser{}
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
	default:
		return nil
	}
}

// ExtractApiKey 根据协议格式从请求头中提取 API Key
func ExtractApiKey(h http.Header, format string) string {
	switch format {
	case FormatAnthropic:
		return h.Get("x-api-key")
	case FormatGemini:
		return h.Get("x-goog-api-key")
	default:
		// openai 和其他兼容格式：Authorization: Bearer xxx
		v := h.Get("Authorization")
		if len(v) > 7 && v[:7] == "Bearer " {
			return v[7:]
		}
		return v
	}
}
