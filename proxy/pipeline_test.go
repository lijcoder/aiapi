package proxy

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/lijcoder/aiapi/proxy/types"
)

type captureWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *captureWriter) Header() http.Header { return w.header }
func (w *captureWriter) WriteStatusCode(statusCode int) {
	w.status = statusCode
}
func (w *captureWriter) Write(body []byte) (int, error) { return w.body.Write(body) }

func TestPipelineSnapshotsGeneratedErrorResponse(t *testing.T) {
	writer := &captureWriter{header: http.Header{}}
	ctx := &types.Context{
		Writer:       writer,
		Err:          errors.New("invalid api key"),
		Code:         types.CodeUnauthorized,
		ErrorMessage: "invalid api key",
	}

	NewPipeline().Execute(ctx)

	if !ctx.ResponseCommitted || ctx.ResponseStatusCode != http.StatusUnauthorized {
		t.Fatalf("response state = committed:%t status:%d", ctx.ResponseCommitted, ctx.ResponseStatusCode)
	}
	if writer.status != http.StatusUnauthorized {
		t.Fatalf("writer status = %d", writer.status)
	}
	if got, want := writer.body.String(), `{"error":{"message":"invalid api key"}}`; got != want {
		t.Fatalf("writer body = %q, want %q", got, want)
	}
	if got, want := string(ctx.RespBody), writer.body.String(); got != want {
		t.Fatalf("cached body = %q, want %q", got, want)
	}
}
