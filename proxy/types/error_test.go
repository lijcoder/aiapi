package types

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestErrorDetailRedactsURLCredentials(t *testing.T) {
	err := fmt.Errorf("forward.go:74: %w", &url.Error{
		Op:  "Post",
		URL: "https://user:pass@example.com/v1/responses?request_id=req_1&api_key=secret&token=another-secret",
		Err: context.Canceled,
	})

	detail := ErrorDetail("upstream request failed", err)
	if strings.Contains(detail, "secret") || strings.Contains(detail, "user:pass") {
		t.Fatalf("error detail leaked credentials: %q", detail)
	}
	if !strings.Contains(detail, "upstream request failed") || !strings.Contains(detail, "context canceled") || !strings.Contains(detail, "request_id=req_1") {
		t.Fatalf("error detail lost useful context: %q", detail)
	}
}
