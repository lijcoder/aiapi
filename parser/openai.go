package parser

import (
	"encoding/json"
	"strings"
)

// OpenAIParser OpenAI 响应解析器
type OpenAIParser struct{}

// openaiNonStreamResp 非流式响应结构
type openaiNonStreamResp struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
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
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIParser) ParseUsage(body []byte) (*Usage, error) {
	var resp openaiNonStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &Usage{
		Provider:     FormatOpenAI,
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		RequestID:    resp.ID,
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
			Provider:     FormatOpenAI,
			Model:        chunk.Model,
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
			TotalTokens:  chunk.Usage.TotalTokens,
			RequestID:    chunk.ID,
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
