package handler

import (
	"io"
	"strings"

	"github.com/lijcoder/aiapi/proxy/sse"
	"github.com/lijcoder/aiapi/proxy/types"
)

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

	var body io.ReadCloser = ctx.HttpResp.Body
	if ctx.P != nil {
		body = sse.NewBody(ctx.HttpResp.Body, ctx.P)
	}

	buf := make([]byte, 512)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := ctx.Writer.Write(buf[:n]); writeErr != nil {
				ctx.Err = writeErr
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			ctx.Err = err
			return
		}
	}
	body.Close()

	if sb, ok := body.(*sse.Body); ok {
		if u := sb.Usage(); u != nil {
			ctx.Usage = u
		}
	}
}

func interceptResponse(ctx *types.Context) {
	if ctx.P == nil {
		return
	}

	body, err := io.ReadAll(ctx.HttpResp.Body)
	if err != nil {
		ctx.Err = err
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
		_, ctx.Err = ctx.Writer.Write(ctx.RespBody)
	}
}
