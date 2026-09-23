package parser

import "testing"

// 非流式响应：usage 中的缓存创建/读取 token 是 input_tokens 之外的增量，
// 统一用量按 OpenAI 语义（input_tokens 为完整输入、cached_tokens 为其子集）映射。
// 取 Key：优先 x-api-key；net/http 会把请求头规范化为 X-Api-Key，因此取值必须大小写不敏感；
// 未带 x-api-key 时回退 Authorization。
func TestAnthropicParser_ParseApiKey(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string][]string
		want    string
	}{
		{"x-api-key", map[string][]string{"X-Api-Key": {"sk-ant"}}, "sk-ant"},
		{"x-api-key 优先于 authorization", map[string][]string{"X-Api-Key": {"sk-ant"}, "Authorization": {"Bearer sk-bearer"}}, "sk-ant"},
		{"x-api-key 为空则继续取下一个头", map[string][]string{"X-Api-Key": {""}, "Authorization": {"Bearer sk-bearer"}}, "sk-bearer"},
		{"x-api-key 带 Bearer 前缀同样会剥掉", map[string][]string{"X-Api-Key": {"Bearer sk-ant"}}, "sk-ant"},
		{"回退 bearer", map[string][]string{"Authorization": {"Bearer sk-bearer"}}, "sk-bearer"},
		{"bearer 前缀大小写不敏感", map[string][]string{"Authorization": {"bearer sk-bearer"}}, "sk-bearer"},
		{"非 bearer 原样返回", map[string][]string{"Authorization": {"sk-raw"}}, "sk-raw"},
		{"无鉴权头", map[string][]string{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Anthropic.ParseApiKey(tc.headers); got != tc.want {
				t.Fatalf("ParseApiKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAnthropicParser_ParseUsage_CacheTokens(t *testing.T) {
	body := []byte(`{
		"id":"msg_01","type":"message","model":"claude-sonnet-4",
		"content":[{"type":"text","text":"hi"}],
		"usage":{"input_tokens":10,"cache_creation_input_tokens":30,"cache_read_input_tokens":60,"output_tokens":5,
			"output_tokens_details":{"thinking_tokens":4}}
	}`)
	u, err := Anthropic.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100 (10 未命中 + 30 缓存创建 + 60 缓存读取)", u.InputTokens)
	}
	if u.CachedTokens != 60 {
		t.Errorf("CachedTokens = %d, want 60", u.CachedTokens)
	}
	if u.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", u.OutputTokens)
	}
	if u.TotalTokens != 105 {
		t.Errorf("TotalTokens = %d, want 105", u.TotalTokens)
	}
	if u.ReasoningTokens != 4 {
		t.Errorf("ReasoningTokens = %d, want 4 (output_tokens_details.thinking_tokens)", u.ReasoningTokens)
	}
	if u.Model != "claude-sonnet-4" || u.RequestID != "msg_01" || u.Provider != FormatAnthropic {
		t.Errorf("identity = provider:%q model:%q request:%q", u.Provider, u.Model, u.RequestID)
	}
}

// 未使用缓存的非流式响应：与旧行为一致。
func TestAnthropicParser_ParseUsage_NoCache(t *testing.T) {
	body := []byte(`{"id":"msg_02","type":"message","model":"claude-haiku","usage":{"input_tokens":7,"output_tokens":3}}`)
	u, err := Anthropic.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.InputTokens != 7 || u.CachedTokens != 0 || u.OutputTokens != 3 || u.TotalTokens != 10 {
		t.Fatalf("unexpected usage: %+v", u)
	}
}

// count_tokens 等非 message 响应不解析为用量，避免误计费。
func TestAnthropicParser_ParseUsage_SkipsNonMessage(t *testing.T) {
	u, err := Anthropic.ParseUsage([]byte(`{"input_tokens":42}`))
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("usage = %+v, want nil", u)
	}
}

// 全部输入命中缓存（Claude Code 等长上下文复用场景）：input_tokens 为 0，
// 输入量只来自缓存读取，映射后缓存量不得超过输入量（计费校验要求）。
func TestAnthropicParser_ParseUsage_FullyCachedInput(t *testing.T) {
	body := []byte(`{"id":"msg_05","type":"message","model":"claude-sonnet-4","usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":1000,"output_tokens":50}}`)
	u, err := Anthropic.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.InputTokens != 1000 || u.CachedTokens != 1000 || u.OutputTokens != 50 || u.TotalTokens != 1050 {
		t.Fatalf("unexpected usage: %+v", u)
	}
	if u.CachedTokens > u.InputTokens {
		t.Fatalf("cached tokens exceed input tokens: %+v", u)
	}
}

// message_start 只提供 identity，不产生用量：单独出现时（无 message_delta）不记用量。
func TestAnthropicParser_ParseStreamUsage_MessageStartOnly(t *testing.T) {
	body := []byte("event: message_start\n" +
		`data: {"type":"message_start","message":{"id":"msg_03","model":"claude-sonnet-4","usage":{"input_tokens":10,"cache_creation_input_tokens":30,"cache_read_input_tokens":60,"output_tokens":1}}}` + "\n\n" +
		"event: message_stop\n" +
		`data: {"type":"message_stop"}` + "\n\n")
	u, err := Anthropic.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("usage = %+v, want nil", u)
	}
}

// message_delta 是唯一的用量来源（累计的输入、缓存、输出、思考 token），
// identity（model / request_id）则来自 message_start。
// message_start 里刻意放了不同的 token 数，用于证明用量不会被它影响。
func TestAnthropicParser_ParseStreamUsage_MessageDelta(t *testing.T) {
	body := []byte(
		"event: message_start\n" +
			`data: {"type":"message_start","message":{"id":"msg_04","model":"claude-sonnet-4","usage":{"input_tokens":7,"cache_read_input_tokens":9,"output_tokens":1}}}` + "\n\n" +
			"event: content_block_delta\n" +
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}` + "\n\n" +
			"event: message_delta\n" +
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":1000,"cache_creation_input_tokens":100,"cache_read_input_tokens":200,"output_tokens":500,"output_tokens_details":{"thinking_tokens":120}}}` + "\n\n" +
			"event: message_stop\n" +
			`data: {"type":"message_stop"}` + "\n\n")

	u, err := Anthropic.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.Provider != FormatAnthropic {
		t.Errorf("Provider = %q, want %q", u.Provider, FormatAnthropic)
	}
	if u.InputTokens != 1300 {
		t.Errorf("InputTokens = %d, want 1300 (1000 未命中 + 100 缓存创建 + 200 缓存读取)", u.InputTokens)
	}
	if u.CachedTokens != 200 {
		t.Errorf("CachedTokens = %d, want 200", u.CachedTokens)
	}
	if u.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", u.OutputTokens)
	}
	if u.ReasoningTokens != 120 {
		t.Errorf("ReasoningTokens = %d, want 120", u.ReasoningTokens)
	}
	if u.TotalTokens != 1800 {
		t.Errorf("TotalTokens = %d, want 1800 (1300 + 500)", u.TotalTokens)
	}
	// identity 来自 message_start（message_delta 事件本身不带 id / model）
	if u.RequestID != "msg_04" || u.Model != "claude-sonnet-4" {
		t.Errorf("identity = request:%q model:%q, want msg_04/claude-sonnet-4", u.RequestID, u.Model)
	}
}

// 老形态上游（如 Bedrock / Vertex 的 Anthropic 兼容流）message_delta 只回 output_tokens：
// 当前只认 message_delta 的口径下输入量记 0；若这类上游在配置里，需要补 message_start 兜底。
func TestAnthropicParser_ParseStreamUsage_MessageDeltaWithoutInput(t *testing.T) {
	body := []byte("event: message_delta\n" +
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":500}}` + "\n\n")
	u, err := Anthropic.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.InputTokens != 0 || u.OutputTokens != 500 || u.TotalTokens != 500 {
		t.Fatalf("unexpected usage: %+v", u)
	}
	if u.Model != "" || u.RequestID != "" {
		t.Fatalf("identity = model:%q request:%q, want empty (无 message_start)", u.Model, u.RequestID)
	}
}

// 流式响应完全没有 usage 事件（如上游异常截断）时不记录用量。
func TestAnthropicParser_ParseStreamUsage_NoUsage(t *testing.T) {
	body := []byte("event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}` + "\n\n")
	u, err := Anthropic.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("usage = %+v, want nil", u)
	}
}
