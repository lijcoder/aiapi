package parser

import (
	"encoding/json"

	"github.com/lijcoder/aiapi/parser/util"
)

// ResponsesParser OpenAI Responses API（/v1/responses）请求/响应解析器。
//
// 与 OpenAIParser（chat/completions）是两套独立的响应形状：
//   - 鉴权、模型提取相同（Bearer / 顶层 model），但 responsess 的 usage 字段为
//     input_tokens / output_tokens（chat 为 prompt_tokens / completion_tokens），
//     流式事件为 response.created / response.output_text.delta / response.completed 等
//     （chat 为 choices[].delta 增量），因此不共用 OpenAIParser 的结构体。
type ResponsesParser struct{}

// responsesRequest 请求体结构（仅用于提取 model）
type responsesRequest struct {
	Model string `json:"model"`
}

func (p *ResponsesParser) ParseModel(body []byte) string {
	var req responsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

// ReplaceModel 替换请求体顶层 model（Responses 的模型名也在顶层）
func (p *ResponsesParser) ReplaceModel(body []byte, name string) ([]byte, error) {
	return util.ReplaceTopLevelModel(body, name)
}

// ParseApiKey 从 Authorization 头取 Bearer token
func (p *ResponsesParser) ParseApiKey(headers map[string][]string) string {
	return util.ExtractBearerToken(headers, HeaderAuthorization)
}

// responsesUsage responses 非流式/流式通用的 usage 结构。
// 注意字段名与 chat 不同（input_tokens 而非 prompt_tokens）。
type responsesUsage struct {
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	TotalTokens        int `json:"total_tokens"`
	InputTokensDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokensDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

// responsesNonStreamResp 非流式响应结构
type responsesNonStreamResp struct {
	ID     string         `json:"id"`
	Object string         `json:"object"`
	Model  string         `json:"model"`
	Usage  responsesUsage `json:"usage"`
}

func (p *ResponsesParser) ParseUsage(body []byte) (*Usage, error) {
	var resp responsesNonStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	// 非 response 对象（如上游的错误体）不解析为用量，避免误计费。
	// 与 anthropic 的 type 判断模式一致。
	if resp.Object != "response" {
		return nil, nil
	}
	return &Usage{
		Provider:        FormatResponses,
		Model:           resp.Model,
		InputTokens:     resp.Usage.InputTokens,
		OutputTokens:    resp.Usage.OutputTokens,
		TotalTokens:     responsesTotalTokens(resp.Usage.TotalTokens, resp.Usage.InputTokens, resp.Usage.OutputTokens),
		RequestID:       resp.ID,
		CachedTokens:    extractResponsesCachedTokens(&resp.Usage),
		ReasoningTokens: extractResponsesReasoningTokens(&resp.Usage),
	}, nil
}

// responsesResponseObj 流式事件中内嵌的 response 对象子集
type responsesResponseObj struct {
	ID    string         `json:"id"`
	Model string         `json:"model"`
	Usage responsesUsage `json:"usage"`
}

// responsesCompletedEvent response.completed 事件：
// 携带完整的 response 对象（id / model / usage），是流式用量与 identity 的唯一来源
type responsesCompletedEvent struct {
	Type     string                `json:"type"`
	Response *responsesResponseObj `json:"response"`
}

// ParseStreamUsage 提取 Responses 流式用量。
//
// 只看 response.completed：它带完整的 response 对象，identity（id / model）与 usage 都在里面。
// 上游给了 total_tokens 就用它，为 0 时按协议口径回退 input + output。
// response.created / output_text.delta / in_progress / failed / incomplete 与用量无关。
func (p *ResponsesParser) ParseStreamUsage(body []byte) (*Usage, error) {
	var usage *Usage
	util.EachSSEData(body, func(data []byte) {
		var evt responsesCompletedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return // 单块解析失败不影响其它块
		}
		if evt.Type != "response.completed" || evt.Response == nil {
			return
		}
		resp := evt.Response
		u := &resp.Usage
		usage = &Usage{
			Provider:        FormatResponses,
			Model:           resp.Model,
			RequestID:       resp.ID,
			InputTokens:     u.InputTokens,
			OutputTokens:    u.OutputTokens,
			TotalTokens:     u.TotalTokens,
			CachedTokens:    extractResponsesCachedTokens(u),
			ReasoningTokens: extractResponsesReasoningTokens(u),
		}
	})
	// 未完成（无 completed 事件）或上游未回传用量时不记零用量
	if usage == nil || (usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0) {
		return nil, nil
	}
	usage.TotalTokens = responsesTotalTokens(usage.TotalTokens, usage.InputTokens, usage.OutputTokens)
	return usage, nil
}

// responsesTotalTokens responses 的 total_tokens 口径：优先用上游值，为 0 时回退 input+output。
// 非流式、流式事件与流式聚合共用同一规则。
func responsesTotalTokens(total, input, output int) int {
	if total > 0 {
		return total
	}
	return input + output
}

// extractResponsesCachedTokens 从 responses usage 中提取缓存命中 token
func extractResponsesCachedTokens(u *responsesUsage) int {
	if u.InputTokensDetails != nil {
		return u.InputTokensDetails.CachedTokens
	}
	return 0
}

// extractResponsesReasoningTokens 从 responses usage 中提取推理 token
func extractResponsesReasoningTokens(u *responsesUsage) int {
	if u.OutputTokensDetails != nil {
		return u.OutputTokensDetails.ReasoningTokens
	}
	return 0
}

// 编译期断言：ResponsesParser 实现 ModelsFormatter。
// responses 自行序列化模型列表（隔离），不依赖 FormatModelList 回退到 OpenAI 格式，
// 后续若 responses 协议推出独立的列表形状，仅需改本实现、不影响其他协议。
var _ ModelsFormatter = (*ResponsesParser)(nil)

// responsesModelList responses「列出模型」响应结构（隔离定义，形状与 OpenAI List Models 一致）。
type responsesModelList struct {
	Object string              `json:"object"`
	Data   []responsesModelObj `json:"data"`
}

type responsesModelObj struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// FormatModels 序列化为 responses 列表格式。
// responses 协议暂无官方模型列表端点，沿用 OpenAI List Models 形状（object="list"）以兼容
// OpenAI SDK 的模型列表解析；结构独立建模，便于按协议演进，而非共享 OpenAI 的结构体。
func (p *ResponsesParser) FormatModels(items []ModelItem) ([]byte, error) {
	list := responsesModelList{Object: "list", Data: make([]responsesModelObj, 0, len(items))}
	for _, it := range items {
		list.Data = append(list.Data, responsesModelObj{
			ID:      it.ID,
			Object:  "model",
			Created: it.CreatedAt.Unix(),
			OwnedBy: it.OwnedBy,
		})
	}
	return json.Marshal(list)
}
