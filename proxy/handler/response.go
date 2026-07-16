package handler

import (
	"bytes"
	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/sse"
	"github.com/lijcoder/aiapi/proxy/types"
	"io"
	"log/slog"
	"strings"
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
func streamResponse(ctx *types.Context) {
	ctx.Stream = true
	for k, vs := range ctx.HttpResp.Header {
		for _, v := range vs {
			ctx.Writer.Header().Add(k, v)
		}
	}
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
	if ctx.P == nil {
		return
	}
	body, err := io.ReadAll(ctx.HttpResp.Body)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "upstream read failed"
		return
	}
	ctx.HttpResp.Body.Close()
	ctx.RespBody = body
	usage, err := ctx.P.ParseUsage(body)
	if err != nil {
		return
	}
	if usage != nil {
		ctx.Usage = usage
	}
}
func writeResponse(ctx *types.Context) {
	for k, vs := range ctx.HttpResp.Header {
		for _, v := range vs {
			ctx.Writer.Header().Add(k, v)
		}
	}
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
