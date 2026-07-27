package handler

import (
	"net/http"
	"testing"

	"github.com/lijcoder/aiapi/proxy/types"
)

// fakeWriter 实现 types.ProxyResponseWrite，仅记录 header
type fakeWriter struct {
	header http.Header
}

func (f *fakeWriter) Header() http.Header            { return f.header }
func (f *fakeWriter) WriteStatusCode(statusCode int) {}
func (f *fakeWriter) Write(body []byte) (int, error) { return len(body), nil }

func TestCopyUpstreamHeaders_FiltersHopByHop(t *testing.T) {
	upstream := http.Header{
		"Content-Type":       {"application/json"},
		"X-Request-Id":       {"req_123"},
		"Connection":         {"keep-alive, X-Internal-Token"},
		"Keep-Alive":         {"timeout=5"},
		"Transfer-Encoding":  {"chunked"},
		"Trailer":            {"X-Foo"},
		"Upgrade":            {"h2c"},
		"Te":                 {"trailers"},
		"Proxy-Authenticate": {"Basic"},
		// 被 Connection 动态点名的头，也应被剥掉
		"X-Internal-Token": {"secret"},
	}

	ctx := &types.Context{
		HttpResp: &http.Response{Header: upstream},
		Writer:   &fakeWriter{header: http.Header{}},
	}
	copyUpstreamHeaders(ctx)

	got := ctx.Writer.Header()

	// 端到端头必须保留
	if got.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be kept, got %q", got.Get("Content-Type"))
	}
	if got.Get("X-Request-Id") != "req_123" {
		t.Errorf("X-Request-Id should be kept, got %q", got.Get("X-Request-Id"))
	}

	// 逐跳头必须剥掉（含 Connection 动态点名的）
	for _, h := range []string{
		"Connection", "Keep-Alive", "Transfer-Encoding", "Trailer",
		"Upgrade", "Te", "Proxy-Authenticate", "X-Internal-Token",
	} {
		if v := got.Get(h); v != "" {
			t.Errorf("hop-by-hop header %s should be stripped, got %q", h, v)
		}
	}
}

func TestCopyUpstreamHeaders_NoConnectionHeader(t *testing.T) {
	upstream := http.Header{
		"Content-Type": {"text/event-stream"},
		"X-Custom":     {"a", "b"}, // 多值头应完整保留
	}

	ctx := &types.Context{
		HttpResp: &http.Response{Header: upstream},
		Writer:   &fakeWriter{header: http.Header{}},
	}
	copyUpstreamHeaders(ctx)

	got := ctx.Writer.Header()
	if got.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type should be kept")
	}
	if vals := got.Values("X-Custom"); len(vals) != 2 {
		t.Errorf("multi-value header should be fully kept, got %v", vals)
	}
}
