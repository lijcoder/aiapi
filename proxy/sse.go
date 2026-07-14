package proxy

import (
	"bytes"
	"io"
	"sync"

	"github.com/lijcoder/aiapi/parser"
)

// 编译期检查 *sseBody 实现了 io.ReadCloser
var _ io.ReadCloser = (*sseBody)(nil)

// sseBody 流式 SSE 响应体包装器，在透传的同时解析事件
type sseBody struct {
	src    io.ReadCloser
	parser parser.Parser

	buf bytes.Buffer // 缓冲不完整的行

	model string
	usage *parser.Usage
	mu    sync.Mutex
}

// newSSEBody 创建 SSE body 包装器
func newSSEBody(src io.ReadCloser, p parser.Parser) *sseBody {
	return &sseBody{
		src:    src,
		parser: p,
	}
}

func (s *sseBody) Read(p []byte) (n int, err error) {
	n, err = s.src.Read(p)
	if n > 0 {
		s.processEvents(p[:n])
	}
	return n, err
}

func (s *sseBody) Close() error {
	return s.src.Close()
}

// Usage 返回解析到的用量（仅在流结束后有效）
func (s *sseBody) Usage() *parser.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

// processEvents 从读取到的数据中提取完整的 SSE 事件并解析
func (s *sseBody) processEvents(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Write(data)
	bufData := s.buf.Bytes()

	sep := []byte("\n\n")
	for {
		idx := bytes.Index(bufData, sep)
		if idx == -1 {
			break
		}
		event := bufData[:idx]
		bufData = bufData[idx+2:]

		s.parseSSEEvent(event)
	}
	s.buf.Reset()
	s.buf.Write(bufData)
}

func (s *sseBody) parseSSEEvent(data []byte) {
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

		event, err := s.parser.ParseStreamEvent(dataContent)
		if err != nil || event == nil {
			continue
		}

		if event.Model != "" && s.model == "" {
			s.model = event.Model
		}
		if event.Usage != nil {
			s.usage = event.Usage
		}
	}
}
