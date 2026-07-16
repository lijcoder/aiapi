package parser

import (
	"encoding/json"
	"strings"
)

// OpenAIParser OpenAI 请求/响应解析器
type OpenAIParser struct{}

// openaiRequest 请求体结构（仅用于提取 model）
type openaiRequest struct {
	Model string `json:"model"`
}

func (p *OpenAIParser) ParseModel(body []byte) string {
	var req openaiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

func (p *OpenAIParser) ParseApiKey(headers map[string][]string) string {
	return extractBearerToken(headers)
}

// openaiUsage 非流式/流式通用的 usage 结构
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	PromptTokensDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}
type openaiNonStreamResp struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage openaiUsage `json:"usage"`
}

// openaiStreamChunk 流式 chunk 结构
type openaiStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Choices []struct {
		Index int    `json:"index"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *openaiUsage `json:"usage"`
}

func (p *OpenAIParser) ParseUsage(body []byte) (*Usage, error) {
	var resp openaiNonStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &Usage{
		Provider:        FormatOpenAI,
		Model:           resp.Model,
		InputTokens:     resp.Usage.PromptTokens,
		OutputTokens:    resp.Usage.CompletionTokens,
		TotalTokens:     resp.Usage.TotalTokens,
		RequestID:       resp.ID,
		CachedTokens:    extractCachedTokens(&resp.Usage),
		ReasoningTokens: extractReasoningTokens(&resp.Usage),
	}, nil
}

func (p *OpenAIParser) ParseStreamEvent(data []byte) (*StreamEvent, error) {
	// 跳过 [DONE]
	line := strings.TrimSpace(string(data))
	if line == "[DONE]" {
		return &StreamEvent{EventType: "done"}, nil
	}

	var chunk openaiStreamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, err
	}

	event := &StreamEvent{}

	// 提取 model
	if chunk.Model != "" {
		event.Model = chunk.Model
	}

	// 提取 content delta
	if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
		event.Content = chunk.Choices[0].Delta.Content
		event.EventType = "content"
	}

	// 提取 usage（最后一条 SSE 事件）
	if chunk.Usage != nil {
		event.Usage = &Usage{
			Provider:        FormatOpenAI,
			Model:           chunk.Model,
			InputTokens:     chunk.Usage.PromptTokens,
			OutputTokens:    chunk.Usage.CompletionTokens,
			TotalTokens:     chunk.Usage.TotalTokens,
			RequestID:       chunk.ID,
			CachedTokens:    extractCachedTokens(chunk.Usage),
			ReasoningTokens: extractReasoningTokens(chunk.Usage),
		}
		event.EventType = "usage"
	}

	// 没有 content 也没有 usage，可能是中间状态事件，跳过
	if event.EventType == "" && chunk.Model == "" {
		return nil, nil
	}

	return event, nil
}

// sseDataLine 从 SSE 行中提取 data 内容
func SSEParseData(line []byte) []byte {
	lineStr := strings.TrimSpace(string(line))
	if lineStr == "" {
		return nil
	}
	if strings.HasPrefix(lineStr, "data: ") {
		return []byte(strings.TrimPrefix(lineStr, "data: "))
	}
	if strings.HasPrefix(lineStr, "data:") {
		return []byte(strings.TrimPrefix(lineStr, "data:"))
	}
	return nil
}

// extractCachedTokens 从 usage 中提取缓存命中 token
func extractCachedTokens(u *openaiUsage) int {
	if u.PromptTokensDetails != nil {
		return u.PromptTokensDetails.CachedTokens
	}
	return 0
}

// extractReasoningTokens 从 usage 中提取推理 token
func extractReasoningTokens(u *openaiUsage) int {
	if u.CompletionTokensDetails != nil {
		return u.CompletionTokensDetails.ReasoningTokens
	}
	return 0
}
