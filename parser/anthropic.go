package parser

import (
	"encoding/json"
	"time"
)

// AnthropicParser Anthropic 请求/响应解析器
type AnthropicParser struct{}

// anthropicRequest 请求体结构（仅用于提取 model）
type anthropicRequest struct {
	Model string `json:"model"`
}

func (p *AnthropicParser) ParseModel(body []byte) string {
	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

func (p *AnthropicParser) ParseApiKey(headers map[string][]string) string {
	if k := headerGet(headers, "x-api-key"); k != "" {
		return k
	}
	return extractBearerToken(headers)
}

// 编译期断言：AnthropicParser 实现 ModelsFormatter
var _ ModelsFormatter = (*AnthropicParser)(nil)

// anthropicModelList Anthropic「列出模型」响应结构
type anthropicModelList struct {
	Data    []anthropicModelObj `json:"data"`
	FirstID *string             `json:"first_id"`
	LastID  *string             `json:"last_id"`
	HasMore bool                `json:"has_more"`
}

type anthropicModelObj struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// FormatModels 序列化为 Anthropic List Models 响应格式。
// 本地全量返回：不支持 limit/before_id/after_id 分页参数，has_more 恒为 false；
// 本地无展示名配置，display_name 以模型名代替。
func (p *AnthropicParser) FormatModels(items []ModelItem) ([]byte, error) {
	list := anthropicModelList{Data: make([]anthropicModelObj, 0, len(items))}
	for _, it := range items {
		list.Data = append(list.Data, anthropicModelObj{
			Type:        "model",
			ID:          it.ID,
			DisplayName: it.ID,
			CreatedAt:   it.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	if len(items) > 0 {
		first, last := items[0].ID, items[len(items)-1].ID
		list.FirstID, list.LastID = &first, &last
	}
	return json.Marshal(list)
}

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

func (p *AnthropicParser) ParseStreamEvent(data []byte) (*StreamEvent, error) {
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
		return &StreamEvent{
			EventType: "usage",
			Usage: &Usage{
				Provider:     FormatAnthropic,
				OutputTokens: evt.Usage.OutputTokens,
			},
		}, nil

	case "message_stop":
		return &StreamEvent{EventType: "done"}, nil

	default:
		// ping / content_block_start / content_block_stop → 跳过
		return nil, nil
	}
}
