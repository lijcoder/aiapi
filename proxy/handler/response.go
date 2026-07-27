package handler

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/sse"
	"github.com/lijcoder/aiapi/proxy/types"
)

// logReadCloser io.TeeReader 的 ReadCloser 包装
type logReadCloser struct {
	io.Reader
	io.Closer
}

// Response 响应处理：流式或非流式
func Response(ctx *types.Context) {
	if ctx.HttpResp == nil {
		return
	}
	if strings.Contains(ctx.HttpResp.Header.Get("Content-Type"), "event-stream") {
		streamResponse(ctx)
	} else {
		interceptResponse(ctx)
		writeResponse(ctx)
	}
}
// hopByHopHeaders RFC 7230 §6.1 定义的逐跳头（key 为 Canonical 形式）。
// 它们只描述「上游 → aiapi」这一段连接的传输状态，对「aiapi → 客户端」
// 这段连接无意义，转发时必须剥掉，由 Go http.Server 自行重新协商。
// （参考标准库 httputil.ReverseProxy 的同款过滤）
var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

// copyUpstreamHeaders 复制上游响应头到客户端，剥掉逐跳头。
// Connection 头里动态点名的头（如 Connection: X-Internal）同样视为逐跳头剥掉。
func copyUpstreamHeaders(ctx *types.Context) {
	src := ctx.HttpResp.Header

	// 解析 Connection 头点名的额外逐跳头（Canonical 形式集合）
	nominated := map[string]bool{}
	for _, v := range src.Values("Connection") {
		for _, name := range strings.Split(v, ",") {
			if name = strings.TrimSpace(name); name != "" {
				nominated[http.CanonicalHeaderKey(name)] = true
			}
		}
	}

	for k, vs := range src {
		if hopByHopHeaders[k] || nominated[k] {
			continue
		}
		for _, v := range vs {
			ctx.Writer.Header().Add(k, v)
		}
	}
}

func streamResponse(ctx *types.Context) {
	ctx.Stream = true
	copyUpstreamHeaders(ctx)
	ctx.Writer.WriteStatusCode(ctx.HttpResp.StatusCode)
	// 用 TeeReader 透传的同时缓存原始响应体用于日志
	var logBuf bytes.Buffer
	logSrc := &logReadCloser{
		Reader: io.TeeReader(ctx.HttpResp.Body, &logBuf),
		Closer: ctx.HttpResp.Body,
	}
	var body io.ReadCloser = logSrc
	if ctx.P != nil {
		body = sse.NewBody(logSrc, ctx.P)
	}
	buf := make([]byte, 512)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := ctx.Writer.Write(buf[:n]); writeErr != nil {
				if isBrokenPipe(writeErr) {
					slog.Warn("client disconnected", "path", ctx.Path)
					return
				}
				ctx.Err = log.WithStack(writeErr)
				ctx.ErrorMessage = "response write failed"
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			ctx.Err = log.WithStack(err)
			ctx.ErrorMessage = "upstream read failed"
			return
		}
	}
	body.Close()
	if sb, ok := body.(*sse.Body); ok {
		if u := sb.Usage(); u != nil {
			ctx.Usage = u
		}
	}
	ctx.RespBody = logBuf.Bytes()
}
func interceptResponse(ctx *types.Context) {
	body, err := io.ReadAll(ctx.HttpResp.Body)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "upstream read failed"
		return
	}
	ctx.HttpResp.Body.Close()
	ctx.RespBody = body

	// 尝试解析用量（不设 ctx.Err，不影响响应转发）
	if ctx.P != nil {
		usage, parseErr := ctx.P.ParseUsage(body)
		if parseErr == nil && usage != nil {
			ctx.Usage = usage
		}
	}
}
func writeResponse(ctx *types.Context) {
	copyUpstreamHeaders(ctx)
	ctx.Writer.WriteStatusCode(ctx.HttpResp.StatusCode)
	if len(ctx.RespBody) > 0 {
		_, err := ctx.Writer.Write(ctx.RespBody)
		if err != nil && !isBrokenPipe(err) {
			ctx.Err = log.WithStack(err)
			ctx.ErrorMessage = "response write failed"
		} else if err != nil {
			slog.Warn("client disconnected", "path", ctx.Path)
		}
	}
}

// isBrokenPipe 判断是否是客户端断连导致的 write 错误
// broken pipe / connection reset by peer 都是正常现象，不视为系统异常
func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "broken pipe") || strings.Contains(s, "connection reset by peer")
}
