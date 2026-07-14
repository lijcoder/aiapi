package parser

import "encoding/json"

// AnthropicParser Anthropic 响应解析器
type AnthropicParser struct{}

// anthropicNonStreamResp 非流式响应结构
type anthropicNonStreamResp struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// anthropicStreamData 流式 data 行通用结构（含 type 字段）
type anthropicStreamData struct {
	Type string `json:"type"`
}

// anthropicMessageStart message_start 事件
type anthropicMessageStart struct {
	Type    string `json:"type"`
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// anthropicContentBlockDelta content_block_delta 事件
type anthropicContentBlockDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// anthropicMessageDelta message_delta 事件
type anthropicMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicParser) ParseUsage(body []byte) (*Usage, error) {
	var resp anthropicNonStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Type != "message" {
		return nil, nil
	}
	return &Usage{
		Provider:     FormatAnthropic,
		Model:        resp.Model,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		RequestID:    resp.ID,
	}, nil
}

func (p *AnthropicParser) ParseStreamEvent(data []byte, cur *Usage) (*StreamEvent, error) {
	// 先解析 type 判断事件类型
	var base anthropicStreamData
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case "message_start":
		var evt anthropicMessageStart
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		return &StreamEvent{
			EventType: "usage",
			Model:     evt.Message.Model,
			Usage: &Usage{
				Provider:     FormatAnthropic,
				Model:        evt.Message.Model,
				InputTokens:  evt.Message.Usage.InputTokens,
				OutputTokens: evt.Message.Usage.OutputTokens,
				TotalTokens:  evt.Message.Usage.InputTokens + evt.Message.Usage.OutputTokens,
				RequestID:    evt.Message.ID,
			},
		}, nil

	case "content_block_delta":
		var evt anthropicContentBlockDelta
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		if evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
			return &StreamEvent{
				EventType: "content",
				Content:   evt.Delta.Text,
			}, nil
		}
		return nil, nil

	case "message_delta":
		var evt anthropicMessageDelta
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		// 合并到当前累计 usage 后返回完整值
		if cur != nil {
			cur.OutputTokens = evt.Usage.OutputTokens
			cur.TotalTokens = cur.InputTokens + cur.OutputTokens
			return &StreamEvent{EventType: "usage", Usage: cur}, nil
		}
		return &StreamEvent{
			EventType: "usage",
			Usage: &Usage{
				Provider:     FormatAnthropic,
				OutputTokens: evt.Usage.OutputTokens,
				TotalTokens:  evt.Usage.OutputTokens,
			},
		}, nil

	case "message_stop":
		return &StreamEvent{EventType: "done"}, nil

	default:
		// ping / content_block_start / content_block_stop → 跳过
		return nil, nil
	}
}
