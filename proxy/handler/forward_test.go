package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/lijcoder/aiapi/proxy/types"
)

// fakeWriter 实现 types.ProxyResponseWrite，捕获 Forward 写入的响应。
type fakeWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
	writes [][]byte
}

func (f *fakeWriter) Header() http.Header { return f.header }
func (f *fakeWriter) WriteStatusCode(statusCode int) {
	f.status = statusCode
}
func (f *fakeWriter) Write(body []byte) (int, error) {
	f.writes = append(f.writes, append([]byte(nil), body...))
	return f.body.Write(body)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type sequenceReadCloser struct {
	chunks     [][]byte
	readIndex  int
	beforeRead func(index int)
	endErr     error
}

func (r *sequenceReadCloser) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		r.beforeRead(r.readIndex)
	}
	if r.readIndex == len(r.chunks) {
		if r.endErr != nil {
			return 0, r.endErr
		}
		return 0, io.EOF
	}
	chunk := r.chunks[r.readIndex]
	r.readIndex++
	return copy(p, chunk), nil
}

func (r *sequenceReadCloser) Close() error { return nil }

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
		"X-Internal-Token":   {"secret"},
	}

	ctx := &types.Context{
		HttpResp: &http.Response{Header: upstream},
		Writer:   &fakeWriter{header: http.Header{}},
	}
	copyUpstreamHeaders(ctx)

	got := ctx.Writer.Header()
	if got.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be kept, got %q", got.Get("Content-Type"))
	}
	if got.Get("X-Request-Id") != "req_123" {
		t.Errorf("X-Request-Id should be kept, got %q", got.Get("X-Request-Id"))
	}
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
		"X-Custom":     {"a", "b"},
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

func TestForwardRelaysAndCachesBufferedResponse(t *testing.T) {
	originalClient := upstreamClient
	upstreamClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("trace") != "yes" {
			t.Errorf("query trace = %q, want yes", r.URL.Query().Get("trace"))
		}
		if r.Header.Get("X-Provider") != "configured" {
			t.Errorf("X-Provider = %q", r.Header.Get("X-Provider"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"model":"test"}` {
			t.Errorf("request body = %q", body)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_123"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"response_1"}`))),
		}, nil
	})}
	defer func() { upstreamClient = originalClient }()

	writer := &fakeWriter{header: http.Header{}}
	ctx := &types.Context{
		Ctx:        context.Background(),
		Method:     http.MethodPost,
		URL:        "http://provider.invalid/v1/test",
		Body:       []byte(`{"model":"test"}`),
		Query:      map[string][]string{"trace": {"yes"}},
		ReqHeaders: map[string][]string{"X-Provider": {"configured"}},
		Writer:     writer,
	}
	Forward(ctx)

	if ctx.Err != nil {
		t.Fatalf("Forward error: %v", ctx.Err)
	}
	if !ctx.ResponseComplete || !ctx.ResponseCommitted || ctx.Stream {
		t.Fatalf("unexpected response state: complete=%t committed=%t stream=%t", ctx.ResponseComplete, ctx.ResponseCommitted, ctx.Stream)
	}
	if writer.status != http.StatusCreated || ctx.ResponseStatusCode != http.StatusCreated {
		t.Fatalf("status writer=%d context=%d", writer.status, ctx.ResponseStatusCode)
	}
	if got, want := writer.body.String(), `{"id":"response_1"}`; got != want {
		t.Fatalf("client body = %q, want %q", got, want)
	}
	if got, want := string(ctx.RespBody), `{"id":"response_1"}`; got != want {
		t.Fatalf("cached body = %q, want %q", got, want)
	}
	if ctx.FirstTokenMs != 0 {
		t.Fatalf("buffered response first token ms = %d, want 0", ctx.FirstTokenMs)
	}
}

func TestForwardRelaysAndCachesStreamResponse(t *testing.T) {
	const body = "data: {\"id\":\"chunk_1\"}\n\ndata: [DONE]\n\n"
	originalClient := upstreamClient
	upstreamClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream; charset=utf-8"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		}, nil
	})}
	defer func() { upstreamClient = originalClient }()

	writer := &fakeWriter{header: http.Header{}}
	ctx := &types.Context{
		Ctx:    context.Background(),
		Method: http.MethodGet,
		URL:    "http://provider.invalid/v1/test",
		Writer: writer,
	}
	Forward(ctx)

	if ctx.Err != nil {
		t.Fatalf("Forward error: %v", ctx.Err)
	}
	if !ctx.ResponseComplete || !ctx.ResponseCommitted || !ctx.Stream {
		t.Fatalf("unexpected response state: complete=%t committed=%t stream=%t", ctx.ResponseComplete, ctx.ResponseCommitted, ctx.Stream)
	}
	if got := writer.body.String(); got != body {
		t.Fatalf("client body = %q, want %q", got, body)
	}
	if got := string(ctx.RespBody); got != body {
		t.Fatalf("cached body = %q, want %q", got, body)
	}
}

func TestForwardStreamDelaysChunksAfterFirstUntilNextRead(t *testing.T) {
	chunks := [][]byte{
		[]byte("first"),
		[]byte("second"),
		[]byte("third"),
	}
	writer := &fakeWriter{header: http.Header{}}
	body := &sequenceReadCloser{
		chunks: chunks,
		beforeRead: func(index int) {
			switch index {
			case 2: // third chunk is about to arrive; second must still be pending.
				if got := len(writer.writes); got != 1 {
					t.Fatalf("writes before third read = %d, want 1", got)
				}
			case 3: // EOF is about to arrive; third must still be pending.
				if got := len(writer.writes); got != 2 {
					t.Fatalf("writes before EOF = %d, want 2", got)
				}
			}
		},
	}

	originalClient := upstreamClient
	upstreamClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       body,
		}, nil
	})}
	defer func() { upstreamClient = originalClient }()

	ctx := &types.Context{
		Ctx:    context.Background(),
		Method: http.MethodGet,
		URL:    "http://provider.invalid/v1/test",
		Writer: writer,
	}
	Forward(ctx)

	if ctx.Err != nil {
		t.Fatalf("Forward error: %v", ctx.Err)
	}
	if !ctx.ResponseComplete {
		t.Fatal("stream should be complete after EOF")
	}
	if got, want := writer.body.String(), "firstsecondthird"; got != want {
		t.Fatalf("client body = %q, want %q", got, want)
	}
	if got, want := string(ctx.RespBody), "firstsecondthird"; got != want {
		t.Fatalf("cached body = %q, want %q", got, want)
	}
}

func TestForwardStreamFlushesPendingChunkBeforeReadError(t *testing.T) {
	readErr := errors.New("upstream connection closed")
	writer := &fakeWriter{header: http.Header{}}
	body := &sequenceReadCloser{
		chunks: [][]byte{[]byte("first"), []byte("last")},
		endErr: readErr,
	}

	originalClient := upstreamClient
	upstreamClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       body,
		}, nil
	})}
	defer func() { upstreamClient = originalClient }()

	ctx := &types.Context{
		Ctx:    context.Background(),
		Method: http.MethodGet,
		URL:    "http://provider.invalid/v1/test",
		Writer: writer,
	}
	Forward(ctx)

	if ctx.Err == nil {
		t.Fatal("Forward error is nil")
	}
	if ctx.ResponseComplete {
		t.Fatal("stream with read error must not be complete")
	}
	if got, want := writer.body.String(), "firstlast"; got != want {
		t.Fatalf("client body = %q, want %q", got, want)
	}
	if got, want := string(ctx.RespBody), "firstlast"; got != want {
		t.Fatalf("cached body = %q, want %q", got, want)
	}
}
