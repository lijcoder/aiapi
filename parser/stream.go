package parser

import (
	"bytes"
	"strings"
)

// ParseStreamUsage 从完整的 SSE 原始响应中提取并合并 usage。
// 无 usage 事件时返回 nil，避免将未要求 usage 的正常流记为零 token 用量。
func ParseStreamUsage(p Parser, body []byte) (*Usage, error) {
	if p == nil {
		return nil, nil
	}
	var result *Usage
	seenTokens := false
	var model, requestID string
	for _, eventData := range splitSSEEvents(body) {
		for _, line := range strings.Split(string(eventData), "\n") {
			data := SSEParseData([]byte(line))
			if data == nil {
				continue
			}
			event, err := p.ParseStreamEvent(data)
			if err != nil || event == nil {
				continue
			}
			if event.Model != "" {
				model = event.Model
			}
			if event.Usage != nil && event.Usage.RequestID != "" {
				requestID = event.Usage.RequestID
			}
			if event.Usage != nil {
				if result == nil {
					result = &Usage{Model: model, RequestID: requestID}
				}
				mergeStreamUsage(result, event.Usage)
			}
			if result != nil && model != "" {
				result.Model = model
			}
			if event.Usage != nil && (event.Usage.InputTokens > 0 || event.Usage.OutputTokens > 0 || event.Usage.TotalTokens > 0) {
				seenTokens = true
			}
		}
	}
	if !seenTokens {
		return nil, nil
	}
	return result, nil
}

func splitSSEEvents(body []byte) [][]byte {
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
	body = bytes.ReplaceAll(body, []byte("\r"), []byte("\n"))
	parts := bytes.Split(body, []byte("\n\n"))
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		return parts[:len(parts)-1]
	}
	return parts
}

func mergeStreamUsage(dst, src *Usage) {
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.RequestID != "" {
		dst.RequestID = src.RequestID
	}
	if src.InputTokens > 0 {
		dst.InputTokens = src.InputTokens
	}
	if src.OutputTokens > 0 {
		dst.OutputTokens = src.OutputTokens
	}
	if src.TotalTokens > 0 {
		dst.TotalTokens = src.TotalTokens
	}
	if src.CachedTokens > 0 {
		dst.CachedTokens = src.CachedTokens
	}
	if src.ReasoningTokens > 0 {
		dst.ReasoningTokens = src.ReasoningTokens
	}
	if dst.TotalTokens == 0 {
		dst.TotalTokens = dst.InputTokens + dst.OutputTokens
	}
}
