package parser

import "testing"

// OpenAI 流式用量：只在带 usage 的块取值，增量块与 [DONE] 直接跳过。
func TestOpenAIParser_ParseStreamUsage_SkipsStreamWithoutUsage(t *testing.T) {
	usage, err := OpenAI.ParseStreamUsage([]byte("data: {\"id\":\"chunk_1\",\"model\":\"gpt-test\",\"choices\":[]}\n\ndata: [DONE]\n\n"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage != nil {
		t.Fatalf("usage = %+v, want nil", usage)
	}
}

// 上游未给 total_tokens 时，按协议口径回退 input + output。
func TestOpenAIParser_ParseStreamUsage_TotalFallback(t *testing.T) {
	body := []byte("data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":20}}\n\n" +
		"data: [DONE]\n\n")
	usage, err := OpenAI.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("usage is nil")
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 20 || usage.TotalTokens != 120 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

// 标准形态：内容增量块不带 usage，最后一块带 usage 并携带 id / model。
func TestOpenAIParser_ParseStreamUsage_FromUsageChunk(t *testing.T) {
	body := []byte(
		"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n" +
			"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cached_tokens\":30},\"completion_tokens_details\":{\"reasoning_tokens\":12}}}\n\n" +
			"data: [DONE]\n\n")

	usage, err := OpenAI.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("usage is nil")
	}
	if usage.Provider != FormatOpenAI || usage.RequestID != "cmpl_1" || usage.Model != "gpt-test" {
		t.Errorf("identity = provider:%q request:%q model:%q", usage.Provider, usage.RequestID, usage.Model)
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 50 || usage.TotalTokens != 150 {
		t.Errorf("tokens = %d/%d/%d, want 100/50/150", usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
	}
	if usage.CachedTokens != 30 || usage.ReasoningTokens != 12 {
		t.Errorf("details = cached %d / reasoning %d, want 30/12", usage.CachedTokens, usage.ReasoningTokens)
	}
}

// 部分网关在每个块都回传累计 usage：以最后一块为准。
func TestOpenAIParser_ParseStreamUsage_LastUsageChunkWins(t *testing.T) {
	body := []byte(
		"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":20,\"total_tokens\":120}}\n\n" +
			"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":40,\"total_tokens\":140}}\n\n")

	usage, err := OpenAI.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("usage is nil")
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 40 || usage.TotalTokens != 140 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

// identity 只从带 usage 的块取：该块没有 id / model 时（非标准上游）两者为空。
func TestOpenAIParser_ParseStreamUsage_IdentityFromUsageChunkOnly(t *testing.T) {
	body := []byte(
		"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":3,\"total_tokens\":10}}\n\n")

	usage, err := OpenAI.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("usage is nil")
	}
	if usage.RequestID != "" || usage.Model != "" {
		t.Fatalf("identity = request:%q model:%q, want empty (usage 块未带 id/model)", usage.RequestID, usage.Model)
	}
	if usage.InputTokens != 7 || usage.OutputTokens != 3 || usage.TotalTokens != 10 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

// 取 Key：Authorization 的 Bearer token；非 Bearer 前缀原样返回（兼容裸 token）。
func TestOpenAIParser_ParseApiKey(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"Bearer sk-abc", "sk-abc"},
		{"bearer sk-abc", "sk-abc"},
		{"sk-abc", "sk-abc"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			headers := map[string][]string{}
			if tc.value != "" {
				headers["Authorization"] = []string{tc.value}
			}
			if got := OpenAI.ParseApiKey(headers); got != tc.want {
				t.Fatalf("ParseApiKey(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
