package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store/model"
)

// errParser 只用于覆盖 ReplaceModel 失败路径
type errParser struct{ err error }

func (p errParser) ParseModel([]byte) string                             { return "gpt-4o" }
func (p errParser) ReplaceModel([]byte, string) ([]byte, error)          { return nil, p.err }
func (p errParser) ParseApiKey(map[string][]string) string               { return "" }
func (p errParser) ParseUsage([]byte) (*parser.Usage, error)             { return nil, nil }
func (p errParser) ParseStreamEvent([]byte) (*parser.StreamEvent, error) { return nil, nil }

func TestRewriteModel_RewritesBodyToProviderModel(t *testing.T) {
	ctx := &types.Context{
		P:         parser.OpenAI,
		Model:     "gpt-4o",
		Body:      []byte(`{"model":"gpt-4o","stream":true}`),
		ModelInfo: &model.Model{Model: "gpt-4o", ProviderModel: "gpt-4o-2024-08-06"},
	}

	RewriteModel(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected error: %v", ctx.Err)
	}
	// 对用户可见的模型名不变（鉴权/计价/日志口径）
	if ctx.Model != "gpt-4o" {
		t.Fatalf("ctx.Model = %q, want gpt-4o", ctx.Model)
	}
	if got := parser.OpenAI.ParseModel(ctx.Body); got != "gpt-4o-2024-08-06" {
		t.Fatalf("upstream model = %q, want gpt-4o-2024-08-06", got)
	}
}

func TestRewriteModel_SkipsWhenTargetEqualsModel(t *testing.T) {
	cases := []struct {
		name string
		info *model.Model
	}{
		{"provider_model empty (legacy row)", &model.Model{Model: "gpt-4o"}},
		{"provider_model blank", &model.Model{Model: "gpt-4o", ProviderModel: "  "}},
		{"provider_model equals model", &model.Model{Model: "gpt-4o", ProviderModel: "gpt-4o"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := []byte(`{"model":"gpt-4o","stream":true}`)
			ctx := &types.Context{P: parser.OpenAI, Model: "gpt-4o", Body: body, ModelInfo: c.info}

			RewriteModel(ctx)

			if ctx.Err != nil {
				t.Fatalf("unexpected error: %v", ctx.Err)
			}
			// 未改写时必须是同一个底层数组（零拷贝、字节零变动）
			if len(ctx.Body) != len(body) || &ctx.Body[0] != &body[0] {
				t.Fatalf("body was rewritten: %q", ctx.Body)
			}
		})
	}
}

func TestRewriteModel_SkipsWithoutParserOrModelInfo(t *testing.T) {
	cases := []struct {
		name string
		ctx  *types.Context
	}{
		{"nil parser", &types.Context{Model: "gpt-4o", Body: []byte(`{"model":"gpt-4o"}`), ModelInfo: &model.Model{Model: "gpt-4o", ProviderModel: "other"}}},
		{"nil model info", &types.Context{P: parser.OpenAI, Model: "gpt-4o", Body: []byte(`{"model":"gpt-4o"}`)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := c.ctx.Body
			RewriteModel(c.ctx)
			if c.ctx.Err != nil {
				t.Fatalf("unexpected error: %v", c.ctx.Err)
			}
			if len(c.ctx.Body) != len(body) || &c.ctx.Body[0] != &body[0] {
				t.Fatalf("body was rewritten: %q", c.ctx.Body)
			}
		})
	}
}

func TestRewriteModel_ReplaceFailureStopsRequest(t *testing.T) {
	body := []byte(`{"model":"gpt-4o"}`)
	ctx := &types.Context{
		P:         errParser{err: errors.New("boom")},
		Model:     "gpt-4o",
		Body:      body,
		ModelInfo: &model.Model{Model: "gpt-4o", ProviderModel: "other"},
	}

	RewriteModel(ctx)

	if ctx.Err == nil {
		t.Fatal("expect error, got nil")
	}
	if ctx.Code != types.CodeUnknown {
		t.Fatalf("code = %v, want CodeUnknown", ctx.Code)
	}
	if ctx.ErrorMessage == "" {
		t.Fatal("expect error message for client")
	}
	// 失败时不改动 body，也不写响应（由 Pipeline 统一输出错误）
	if &ctx.Body[0] != &body[0] || ctx.ResponseCommitted {
		t.Fatalf("body/response mutated: %q committed=%v", ctx.Body, ctx.ResponseCommitted)
	}
}

// TestRewriteModel_ThenForwardUpstream 链路组合验证：改写后的请求体就是发往上游的内容，
// 上游响应原样透传（响应里的 model 不改写）
func TestRewriteModel_ThenForwardUpstream(t *testing.T) {
	originalClient := upstreamClient
	defer func() { upstreamClient = originalClient }()
	var upstreamBody []byte
	upstreamClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		upstreamBody, _ = io.ReadAll(r.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"model":"gpt-4o-2024-08-06","choices":[]}`)),
		}, nil
	})}

	writer := &fakeWriter{header: http.Header{}}
	ctx := &types.Context{
		Ctx:       context.Background(),
		Method:    http.MethodPost,
		URL:       "http://provider.invalid/v1/chat/completions",
		Body:      []byte(`{"model":"gpt-4o","stream":false}`),
		Model:     "gpt-4o",
		ModelInfo: &model.Model{Model: "gpt-4o", ProviderModel: "gpt-4o-2024-08-06"},
		P:         parser.OpenAI,
		Writer:    writer,
	}

	RewriteModel(ctx)
	Forward(ctx)

	if ctx.Err != nil {
		t.Fatalf("unexpected error: %v", ctx.Err)
	}
	if got := parser.OpenAI.ParseModel(upstreamBody); got != "gpt-4o-2024-08-06" {
		t.Fatalf("upstream received model %q, body=%s", got, upstreamBody)
	}
	if ctx.Model != "gpt-4o" {
		t.Fatalf("ctx.Model = %q, want gpt-4o", ctx.Model)
	}
	// 响应体（含 model 字段）原样回给客户端
	if got, want := writer.body.String(), `{"model":"gpt-4o-2024-08-06","choices":[]}`; got != want {
		t.Fatalf("client body = %q, want %q", got, want)
	}
}
