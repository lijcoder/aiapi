package handler

import (
	"net/http"
	"testing"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
)

func TestParseUsage_NonStream(t *testing.T) {
	ctx := &types.Context{
		P:                parser.OpenAI,
		HttpResp:         &http.Response{StatusCode: http.StatusOK},
		RespBody:         []byte(`{"id":"cmpl_1","model":"gpt-test","usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`),
		ResponseComplete: true,
	}
	ParseUsage(ctx)
	if ctx.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ctx.Usage.InputTokens != 3 || ctx.Usage.OutputTokens != 2 || ctx.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", ctx.Usage)
	}
}

// OpenAI 流式：用量只在带 usage 的块取（该块同时带 id / model）。
func TestParseUsage_OpenAIStream(t *testing.T) {
	ctx := &types.Context{
		P:                parser.OpenAI,
		HttpResp:         &http.Response{StatusCode: http.StatusOK},
		Stream:           true,
		ResponseComplete: true,
		RespBody: []byte("data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n" +
			"data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
			"data: [DONE]\n\n"),
	}
	ParseUsage(ctx)
	if ctx.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ctx.Usage.Model != "gpt-test" || ctx.Usage.RequestID != "cmpl_1" {
		t.Fatalf("identity = model:%q request:%q, want gpt-test/cmpl_1", ctx.Usage.Model, ctx.Usage.RequestID)
	}
	if ctx.Usage.InputTokens != 3 || ctx.Usage.OutputTokens != 2 || ctx.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", ctx.Usage)
	}
}

// Anthropic 流式：用量只认 message_delta（累计的输入、缓存、输出、思考 token 都在这里）。
// 缓存创建/读取并入完整输入量，思考 token 单独统计，总量按协议口径重算。
func TestParseUsage_AnthropicStream(t *testing.T) {
	ctx := &types.Context{
		P:                parser.Anthropic,
		HttpResp:         &http.Response{StatusCode: http.StatusOK},
		Stream:           true,
		ResponseComplete: true,
		RespBody: []byte("event: message_start\n" +
			`data: {"type":"message_start","message":{"id":"msg_1","model":"claude-sonnet-4","usage":{"input_tokens":1000,"cache_creation_input_tokens":100,"cache_read_input_tokens":200,"output_tokens":1}}}` + "\n\n" +
			"event: message_delta\n" +
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":1000,"cache_creation_input_tokens":100,"cache_read_input_tokens":200,"output_tokens":400,"output_tokens_details":{"thinking_tokens":90}}}` + "\n\n" +
			"event: message_stop\n" +
			`data: {"type":"message_stop"}` + "\n\n"),
	}
	ParseUsage(ctx)
	if ctx.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ctx.Usage.InputTokens != 1300 || ctx.Usage.CachedTokens != 200 || ctx.Usage.OutputTokens != 400 || ctx.Usage.TotalTokens != 1700 {
		t.Fatalf("unexpected usage: %+v", ctx.Usage)
	}
	if ctx.Usage.ReasoningTokens != 90 {
		t.Fatalf("ReasoningTokens = %d, want 90", ctx.Usage.ReasoningTokens)
	}
	// identity 取自 message_start（用量只取 message_delta）
	if ctx.Usage.RequestID != "msg_1" || ctx.Usage.Model != "claude-sonnet-4" {
		t.Fatalf("identity = request:%q model:%q, want msg_1/claude-sonnet-4", ctx.Usage.RequestID, ctx.Usage.Model)
	}
	// 计费校验要求 cached_tokens <= input_tokens，映射后必须始终成立。
	if ctx.Usage.CachedTokens > ctx.Usage.InputTokens {
		t.Fatalf("cached tokens exceed input tokens: %+v", ctx.Usage)
	}
}

func TestParseUsage_SkipsIncompleteOrErrorResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		full   bool
	}{
		{name: "incomplete", status: http.StatusOK, full: false},
		{name: "error", status: http.StatusBadGateway, full: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &types.Context{
				P:                parser.OpenAI,
				HttpResp:         &http.Response{StatusCode: tc.status},
				RespBody:         []byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`),
				ResponseComplete: tc.full,
			}
			ParseUsage(ctx)
			if ctx.Usage != nil {
				t.Fatalf("Usage = %+v, want nil", ctx.Usage)
			}
		})
	}
}
