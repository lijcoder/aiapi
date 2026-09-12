package types

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrorDetail 返回适合写入日志和 request_logs 的错误详情。
// message 标识业务错误类型，err 保留底层异常；上游请求错误包含的 URL 会脱敏。
func ErrorDetail(message string, err error) string {
	if err == nil {
		return ""
	}
	detail := err.Error()
	var requestErr *url.Error
	if errors.As(err, &requestErr) {
		detail = fmt.Sprintf("%s %q: %v", requestErr.Op, redactURL(requestErr.URL), requestErr.Err)
	}
	if message == "" {
		return detail
	}
	return message + ": " + detail
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = url.User("***")
	}
	q := u.Query()
	for key := range q {
		if isSensitiveName(key) {
			q[key] = []string{"***"}
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func isSensitiveName(name string) bool {
	low := strings.ToLower(name)
	for _, keyword := range []string{"auth", "cookie", "key", "token", "secret", "password"} {
		if strings.Contains(low, keyword) {
			return true
		}
	}
	return false
}
