package parser

import (
	"encoding/json"
	"time"

	"github.com/lijcoder/aiapi/parser/util"
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

// ReplaceModel 替换请求体顶层 model（Anthropic 的模型名也在顶层）
func (p *AnthropicParser) ReplaceModel(body []byte, name string) ([]byte, error) {
	return util.ReplaceTopLevelModel(body, name)
}

// anthropicAPIKeyHeader Anthropic 专用鉴权头，优先于标准 Authorization
const anthropicAPIKeyHeader = "x-api-key"

// ParseApiKey 按 Anthropic 的鉴权头优先级取值：x-api-key → Authorization
func (p *AnthropicParser) ParseApiKey(headers map[string][]string) string {
	return util.ExtractBearerToken(headers, anthropicAPIKeyHeader, HeaderAuthorization)
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

// anthropicUsage Anthropic usage 结构（非流式响应与流式 message_delta 事件共用）。
//
// 与 OpenAI 语义的差异：Anthropic 的 input_tokens 只统计未命中缓存的输入，
// 缓存创建（cache_creation_input_tokens）与缓存读取（cache_read_input_tokens）
// 是单独的计数，三者相加才是完整输入量；流式下这些字段都是本次请求的累计值。
type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	// OutputTokensDetails 输出分解：thinking_tokens 是模型内部推理消耗（含思考块分隔符），
	// 为 output_tokens 的子集，计费仍以 output_tokens 为准，这里只用于用量统计。
	OutputTokensDetails *struct {
		ThinkingTokens int `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
}

// toUsage 把 Anthropic usage 映射为统一用量（OpenAI 语义：input_tokens 为完整输入、
// cached_tokens 是其子集），供计费与用量统计使用：
//   - input_tokens = 未命中输入 + 缓存创建 + 缓存读取（完整输入量）
//   - cached_tokens = 缓存读取（按缓存命中价计费；缓存创建并入未命中价计费，
//     统一用量没有独立的缓存写入价）
//   - reasoning_tokens = output_tokens_details.thinking_tokens（output 的子集）
//   - total_tokens = 完整输入 + 输出
//
// identity（model / request_id）不在这里设置：流式 message_delta 事件不携带它们，
// 由调用方按各自能拿到的信息补充。
func (u anthropicUsage) toUsage() *Usage {
	input := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	usage := &Usage{
		Provider:        FormatAnthropic,
		InputTokens:     input,
		OutputTokens:    u.OutputTokens,
		TotalTokens:     anthropicTotalTokens(input, u.OutputTokens),
		CachedTokens:    u.CacheReadInputTokens,
		ReasoningTokens: u.thinkingTokens(),
	}
	return usage
}

// thinkingTokens 取输出分解里的思考 token（无该字段时上游未提供，记 0）
func (u anthropicUsage) thinkingTokens() int {
	if u.OutputTokensDetails == nil {
		return 0
	}
	return u.OutputTokensDetails.ThinkingTokens
}

// anthropicTotalTokens Anthropic 的总量口径：完整输入 + 输出（上游 usage 不提供 total_tokens）。
func anthropicTotalTokens(input, output int) int {
	return input + output
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
	Usage anthropicUsage `json:"usage"`
}

// anthropicStreamEvent 流式 data 行的 type 字段，用于分发事件
type anthropicStreamEvent struct {
	Type string `json:"type"`
}

// anthropicMessageStart message_start 事件：只取 identity（message id / model）。
// 这里刻意不建模 usage——用量一律来自 message_delta，message_start 的 output_tokens
// 只是首个很小的计数，input 侧计数与 message_delta 同源。
type anthropicMessageStart struct {
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
	} `json:"message"`
}

// anthropicMessageDelta message_delta 事件：本次请求的累计用量都在这里。
// 官方 API 的 message_delta.usage 包含 input_tokens / cache_creation_input_tokens /
// cache_read_input_tokens / output_tokens 与 output_tokens_details.thinking_tokens 全部字段。
type anthropicMessageDelta struct {
	Usage anthropicUsage `json:"usage"`
}

func (p *AnthropicParser) ParseUsage(body []byte) (*Usage, error) {
	var resp anthropicNonStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	// 非 message 响应（如 /v1/messages/count_tokens 的 {input_tokens}）不解析为用量，
	// 避免误计费。
	if resp.Type != "message" {
		return nil, nil
	}
	usage := resp.Usage.toUsage()
	usage.Model, usage.RequestID = resp.Model, resp.ID
	return usage, nil
}

// ParseStreamUsage 提取 Anthropic 流式用量。
//
// 事件分工：
//   - message_start：只取 identity（message id / model）
//   - message_delta：只取用量，且是本次请求的累计值
//     （input_tokens + 缓存创建 + 缓存读取 + output_tokens + thinking_tokens）
//
// message_start 的 output_tokens 只是首个很小的计数，input 侧计数与 message_delta 同源，
// 因此用量不从它取；两事件顺序固定（start 在前），identity 在遍历结束后统一回填。
func (p *AnthropicParser) ParseStreamUsage(body []byte) (*Usage, error) {
	var (
		usage      *Usage
		seenTokens bool
		model      string
		requestID  string
	)
	util.EachSSEData(body, func(data []byte) {
		var base anthropicStreamEvent
		if err := json.Unmarshal(data, &base); err != nil {
			return // 单块解析失败不影响其它块
		}
		switch base.Type {
		case "message_start":
			var evt anthropicMessageStart
			if err := json.Unmarshal(data, &evt); err != nil {
				return
			}
			if evt.Message.ID != "" {
				requestID = evt.Message.ID
			}
			if evt.Message.Model != "" {
				model = evt.Message.Model
			}
		case "message_delta":
			var evt anthropicMessageDelta
			if err := json.Unmarshal(data, &evt); err != nil {
				return
			}
			usage = evt.Usage.toUsage()
			if usage.InputTokens > 0 || usage.OutputTokens > 0 {
				seenTokens = true
			}
		default:
			// ping / content_block_* / message_stop 与用量解析无关
		}
	})
	// 无 message_delta（上游异常截断等）时不记用量
	if usage == nil || !seenTokens {
		return nil, nil
	}
	usage.Model, usage.RequestID = model, requestID
	return usage, nil
}
