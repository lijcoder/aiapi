package util

import "strings"

// ExtractBearerToken 按给定顺序依次查找请求头，返回第一个非空值对应的 token：
//   - 值带 "Bearer " 前缀（大小写不敏感）时剥掉前缀
//   - 没有该前缀时原样返回（兼容裸 token）
//   - 所有头都没值（或没传头名）时返回空串
//
// 头名与优先级由协议决定，本函数不假设任何具体头（标准头常量见 parser.HeaderAuthorization）。
func ExtractBearerToken(headers map[string][]string, names ...string) string {
	for _, name := range names {
		if v := headerGet(headers, name); v != "" {
			return trimBearerPrefix(v)
		}
	}
	return ""
}

// headerGet 从 map[string][]string 头部中获取指定 key 的第一个值（不区分大小写）。
// net/http 会把请求头规范化为 CanonicalMIMEHeaderKey（如 X-Api-Key），故必须大小写不敏感。
func headerGet(headers map[string][]string, key string) string {
	lower := strings.ToLower(key)
	for k, vs := range headers {
		if strings.ToLower(k) == lower && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// trimBearerPrefix 剥掉 "Bearer " 前缀（大小写不敏感），无该前缀时原样返回。
func trimBearerPrefix(v string) string {
	if len(v) > 7 && strings.EqualFold(v[:7], "bearer ") {
		return v[7:]
	}
	return v
}
