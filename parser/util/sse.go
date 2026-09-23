package util

import (
	"bytes"
	"strings"
)

// SSEParseData 从 SSE 行中提取 data 内容。
// 非 data 行（event:/id:/注释/空行）返回 nil，由调用方跳过。
func SSEParseData(line []byte) []byte {
	lineStr := strings.TrimSpace(string(line))
	if lineStr == "" {
		return nil
	}
	if strings.HasPrefix(lineStr, "data: ") {
		return []byte(strings.TrimPrefix(lineStr, "data: "))
	}
	if strings.HasPrefix(lineStr, "data:") {
		return []byte(strings.TrimPrefix(lineStr, "data:"))
	}
	return nil
}

// SplitSSEEvents 把完整 SSE 响应切成事件块（以空行分隔），兼容 LF / CRLF / CR 换行。
func SplitSSEEvents(body []byte) [][]byte {
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
	body = bytes.ReplaceAll(body, []byte("\r"), []byte("\n"))
	parts := bytes.Split(body, []byte("\n\n"))
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		return parts[:len(parts)-1]
	}
	return parts
}

// EachSSEData 依次回调 SSE 响应中每行的 data 载荷（已剥离 "data:" 前缀与首尾空白）。
//
// 事件切分与 data 行提取属于 SSE 标准行为，各协议共用；载荷的事件语义
// （JSON 结构、字段合并口径、总量口径）由各协议自行处理，本函数不做任何解析。
// 非 data 行与空载荷不回调。
func EachSSEData(body []byte, fn func(data []byte)) {
	for _, eventData := range SplitSSEEvents(body) {
		for _, line := range strings.Split(string(eventData), "\n") {
			data := SSEParseData([]byte(line))
			if len(data) == 0 {
				continue
			}
			fn(data)
		}
	}
}
