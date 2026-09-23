package parser

import (
	"encoding/json"

	"github.com/lijcoder/aiapi/parser/util"
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

// ReplaceModel 替换请求体顶层 model（OpenAI 系协议的模型名都在顶层）
func (p *OpenAIParser) ReplaceModel(body []byte, name string) ([]byte, error) {
	return util.ReplaceTopLevelModel(body, name)
}

// ParseApiKey 从 Authorization 头取 Bearer token
func (p *OpenAIParser) ParseApiKey(headers map[string][]string) string {
	return util.ExtractBearerToken(headers, HeaderAuthorization)
}

// 编译期断言：OpenAIParser 实现 ModelsFormatter
var _ ModelsFormatter = (*OpenAIParser)(nil)

// openaiModelList 「列出模型」响应结构
type openaiModelList struct {
	Object string           `json:"object"`
	Data   []openaiModelObj `json:"data"`
}

type openaiModelObj struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// FormatModels 序列化为 OpenAI List Models 响应格式
func (p *OpenAIParser) FormatModels(items []ModelItem) ([]byte, error) {
	list := openaiModelList{Object: "list", Data: make([]openaiModelObj, 0, len(items))}
	for _, it := range items {
		list.Data = append(list.Data, openaiModelObj{
			ID:      it.ID,
			Object:  "model",
			Created: it.CreatedAt.Unix(),
			OwnedBy: it.OwnedBy,
		})
	}
	return json.Marshal(list)
}

// openaiUsage 非流式/流式通用的 usage 结构
type openaiUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
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

// openaiStreamUsageChunk 流式用量解析所需的 chunk 子集。
// choices[] 的增量内容与用量无关（响应已原样透传给客户端），这里只取 identity 与 usage。
type openaiStreamUsageChunk struct {
	ID    string       `json:"id"`
	Model string       `json:"model"`
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

// ParseStreamUsage 提取 OpenAI 流式用量。
//
// chat/completions 只在最后一块给出 usage（stream_options.include_usage），该块同时
// 携带本次请求的 id / model，因此只在带 usage 的块取值，不跨块合并。
// 上游未给 total_tokens 时按协议口径回退 input + output。
func (p *OpenAIParser) ParseStreamUsage(body []byte) (*Usage, error) {
	var usage *Usage
	util.EachSSEData(body, func(data []byte) {
		var chunk openaiStreamUsageChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			return // [DONE] 等非 JSON 块直接跳过
		}
		if chunk.Usage == nil {
			return // 内容增量块与用量无关
		}
		// 上游若在多个块回传累计 usage（部分网关行为），后到者覆盖前值
		usage = &Usage{
			Provider:        FormatOpenAI,
			Model:           chunk.Model,
			RequestID:       chunk.ID,
			InputTokens:     chunk.Usage.PromptTokens,
			OutputTokens:    chunk.Usage.CompletionTokens,
			TotalTokens:     chunk.Usage.TotalTokens,
			CachedTokens:    extractCachedTokens(chunk.Usage),
			ReasoningTokens: extractReasoningTokens(chunk.Usage),
		}
	})
	// 未要求用量（无 include_usage）或上游未回传时不记零用量
	if usage == nil || (usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0) {
		return nil, nil
	}
	usage.TotalTokens = openaiTotalTokens(usage.TotalTokens, usage.InputTokens, usage.OutputTokens)
	return usage, nil
}

// openaiTotalTokens OpenAI 的 total_tokens 口径：优先用上游值，为 0 时回退 input+output。
func openaiTotalTokens(total, input, output int) int {
	if total > 0 {
		return total
	}
	return input + output
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
