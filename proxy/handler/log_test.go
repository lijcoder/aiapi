package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
)

func TestIsSensitiveHeader(t *testing.T) {
	sensitive := []string{
		"Authorization",       // 标准凭证头
		"Proxy-Authorization", // 代理凭证
		"X-Api-Key",           // OpenAI 风格
		"api-key",             // Azure 风格
		"X-Goog-Api-Key",      // Gemini
		"X-Auth-Token",        // 常见网关
		"Cookie",              // 会话
		"Set-Cookie",          // 会话
		"X-Refresh-Token",     // token 类
		"X-Client-Secret",     // secret 类
		"X-Api-Password",      // password 类
		"x-custom-API-KEY",    // 大小写混合
	}
	for _, h := range sensitive {
		if !isSensitiveHeader(h) {
			t.Errorf("%s should be masked", h)
		}
	}

	normal := []string{
		"Content-Type",
		"Accept",
		"User-Agent",
		"X-Request-Id",
		"OpenAI-Organization",
		"Accept-Encoding",
	}
	for _, h := range normal {
		if isSensitiveHeader(h) {
			t.Errorf("%s should NOT be masked", h)
		}
	}
}

func TestRequestLogErrorAndStatusAfterResponseWriteFailure(t *testing.T) {
	ctx := &types.Context{
		Err: log.WithStack(&url.Error{
			Op:  "Post",
			URL: "https://provider.example/v1/responses?api_key=secret",
			Err: context.Canceled,
		}),
		Code:               types.CodeUnknown,
		ErrorMessage:       "provider stream read failed",
		ResponseStatusCode: http.StatusOK,
	}

	if got := requestLogStatusCode(ctx); got != http.StatusOK {
		t.Fatalf("status code = %d, want %d", got, http.StatusOK)
	}
	detail := types.ErrorDetail(ctx.ErrorMessage, ctx.Err)
	if strings.Contains(detail, "secret") || !strings.Contains(detail, "provider stream read failed") || !strings.Contains(detail, "context canceled") {
		t.Fatalf("unexpected error detail: %q", detail)
	}
}

func TestRequestLogStatusCodeUsesResponseStatus(t *testing.T) {
	ctx := &types.Context{ResponseStatusCode: http.StatusBadGateway}
	if got := requestLogStatusCode(ctx); got != http.StatusBadGateway {
		t.Fatalf("status code = %d, want %d", got, http.StatusBadGateway)
	}
	ctx.ResponseStatusCode = 0
	ctx.HttpResp = &http.Response{StatusCode: http.StatusUnauthorized}
	if got := requestLogStatusCode(ctx); got != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", got, http.StatusUnauthorized)
	}
}
