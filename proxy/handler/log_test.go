package handler

import "testing"

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
