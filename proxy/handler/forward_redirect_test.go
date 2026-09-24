package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/lijcoder/aiapi/proxy/types"
)

// TestUpstreamClientRefusesRedirects 是凭证泄漏的回归测试：上游返回 3xx 时不得跟随。
//
// 标准库只在跨域重定向时剥离 Authorization / Www-Authenticate / Cookie / Cookie2，
// 而本项目的上游凭证来自 providers.config.headers，常见形态是 Anthropic 的 x-api-key、
// Azure 的 api-key——这些自定义头若跟随重定向会被原样发给重定向目标（等于交出上游密钥）。
// 本用例同时断言"重定向目标未被访问"和"3xx 原样透传给客户端"。
func TestUpstreamClientRefusesRedirects(t *testing.T) {
	var targetHits int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&targetHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("解析重定向目标失败: %v", err)
	}
	// 刻意的跨域重定向：origin 是 127.0.0.1，目标是 localhost（host 不同），
	// 未设 CheckRedirect 时标准库会真的发起第二次请求，本用例即会失败。
	redirectTo := "http://localhost:" + targetURL.Port() + "/v1/messages"

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTo, http.StatusFound)
	}))
	defer origin.Close()

	writer := &fakeWriter{header: http.Header{}}
	ctx := &types.Context{
		Ctx:        context.Background(),
		Method:     http.MethodPost,
		URL:        origin.URL + "/v1/messages",
		Body:       []byte(`{"model":"test"}`),
		ReqHeaders: map[string][]string{"x-api-key": {"sk-upstream-secret"}},
		Writer:     writer,
	}
	Forward(ctx)

	if ctx.Err != nil {
		t.Fatalf("Forward 返回错误: %v", ctx.Err)
	}
	if got := atomic.LoadInt32(&targetHits); got != 0 {
		t.Fatalf("重定向目标被访问 %d 次，期望 0 次（上游凭证会泄漏到该目标）", got)
	}
	if writer.status != http.StatusFound || ctx.ResponseStatusCode != http.StatusFound {
		t.Fatalf("状态码 writer=%d context=%d，期望 302（3xx 应原样透传）", writer.status, ctx.ResponseStatusCode)
	}
}

// TestUpstreamClientRedirectPolicyConfigured 断言转发 client 显式配置了重定向策略，
// 防止后续有人删除 CheckRedirect 让凭证重新暴露（配置存在性检查，不发起请求）。
func TestUpstreamClientRedirectPolicyConfigured(t *testing.T) {
	if upstreamClient.CheckRedirect == nil {
		t.Fatal("upstreamClient 必须设置 CheckRedirect，否则上游 3xx 会带着凭证被跟随")
	}
	if err := upstreamClient.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatalf("CheckRedirect 返回 %v，期望 http.ErrUseLastResponse", err)
	}
}
