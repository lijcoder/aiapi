package sse

import (
	"bytes"
	"io"
	"sync"

	"github.com/lijcoder/aiapi/parser"
)

// Body 流式 SSE 响应体包装器，在透传的同时解析事件
type Body struct {
	src    io.ReadCloser
	parser parser.Parser

	buf bytes.Buffer

	model string
	usage *parser.Usage
	mu    sync.Mutex
}

// NewBody 创建 SSE body 包装器
func NewBody(src io.ReadCloser, p parser.Parser) *Body {
	return &Body{src: src, parser: p, usage: &parser.Usage{}}
}

func (b *Body) Read(p []byte) (n int, err error) {
	n, err = b.src.Read(p)
	if n > 0 {
		b.processEvents(p[:n])
	}
	return n, err
}

func (b *Body) Close() error {
	return b.src.Close()
}

// Usage 返回解析到的用量（仅在流结束后有效）
func (b *Body) Usage() *parser.Usage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.usage
}

func (b *Body) processEvents(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf.Write(data)
	bufData := b.buf.Bytes()

	sep := []byte("\n\n")
	for {
		idx := bytes.Index(bufData, sep)
		if idx == -1 {
			break
		}
		event := bufData[:idx]
		bufData = bufData[idx+2:]

		b.parseSSEEvent(event)
	}
	b.buf.Reset()
	b.buf.Write(bufData)
}

func (b *Body) parseSSEEvent(data []byte) {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		dataContent := parser.SSEParseData(line)
		if dataContent == nil {
			continue
		}

		event, err := b.parser.ParseStreamEvent(dataContent)
		if err != nil || event == nil {
			continue
		}

		if event.Model != "" && b.model == "" {
			b.model = event.Model
		}
		if event.Usage != nil {
			// 累加：非零字段覆盖，零值保留（OpenAI 全量覆盖，Anthropic 跨事件累加）
			if event.Usage.InputTokens > 0 {
				b.usage.InputTokens = event.Usage.InputTokens
			}
			if event.Usage.OutputTokens > 0 {
				b.usage.OutputTokens = event.Usage.OutputTokens
			}
			if event.Usage.CachedTokens > 0 {
				b.usage.CachedTokens = event.Usage.CachedTokens
			}
			if event.Usage.ReasoningTokens > 0 {
				b.usage.ReasoningTokens = event.Usage.ReasoningTokens
			}
			if event.Usage.Model != "" {
				b.usage.Model = event.Usage.Model
			}
			if event.Usage.RequestID != "" {
				b.usage.RequestID = event.Usage.RequestID
			}
			if event.Usage.TotalTokens > 0 {
				b.usage.TotalTokens = event.Usage.TotalTokens
			} else {
				b.usage.TotalTokens = b.usage.InputTokens + b.usage.OutputTokens
			}
		}
	}
}
