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

func TestParseUsage_StreamMergesEvents(t *testing.T) {
	ctx := &types.Context{
		P:                parser.OpenAI,
		HttpResp:         &http.Response{StatusCode: http.StatusOK},
		Stream:           true,
		ResponseComplete: true,
		RespBody: []byte("data: {\"id\":\"cmpl_1\",\"model\":\"gpt-test\",\"choices\":[]}\n\n" +
			"data: {\"id\":\"cmpl_1\",\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
			"data: [DONE]\n\n"),
	}
	ParseUsage(ctx)
	if ctx.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ctx.Usage.Model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", ctx.Usage.Model)
	}
	if ctx.Usage.InputTokens != 3 || ctx.Usage.OutputTokens != 2 || ctx.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", ctx.Usage)
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
