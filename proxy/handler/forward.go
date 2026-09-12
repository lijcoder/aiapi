package handler

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
)

// upstreamClient 转发上游专用 client，进程内共享（Transport 内部按 host 池化连接）。
//
// 不用 http.DefaultClient 的原因：
//   - 默认 MaxIdleConnsPerHost=2，高并发下同 provider 连接频繁销毁重建（TCP+TLS 握手开销）
//   - 无法独立调优，且与进程内其他共用 DefaultClient 的代码互相影响
//
// 超时设计：
//   - 不设 Client.Timeout / ResponseHeaderTimeout：LLM 流式响应时长不可预估
//     （慢模型可能数分钟才吐第一个 token），任何固定死线都会误杀健康长流
//   - 连接生命周期由请求 context 控制（客户端断开即取消，见 NewRequestWithContext）
//   - 只对「建连/TLS 握手」这类有合理上限的阶段设超时
var upstreamClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // TCP 建连超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   64, // 同一 provider 域名下保留的空闲连接（默认仅 2）
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// Forward 发起上游请求并透传响应。
//
// 该 handler 只负责 HTTP 传输：构造请求、复制响应头、写回客户端，并缓存
// 上游返回的原始 body。协议解析由后续 ParseUsage handler 负责。
func Forward(ctx *types.Context) {
	// 注意：body 必须直接传 *bytes.Reader，不能用 io.NopCloser 包装。
	// NewRequest 内部对 *bytes.Reader 做类型断言以设置 ContentLength 和 GetBody；
	// 包装后断言失败 → ContentLength=0 → 走 chunked 编码，且连接无法复用。
	// 携带请求 context：客户端断开时取消上游请求，及时释放连接与 goroutine；
	// 流式阶段 body.Read 也会被取消，透传循环随之退出。
	req, err := http.NewRequestWithContext(ctx.Ctx, ctx.Method, ctx.URL, bytes.NewReader(ctx.Body))
	if err != nil {
		setForwardError(ctx, err, types.InternalServerError)
		return
	}
	req.Header = make(http.Header)
	for k, vs := range ctx.ReqHeaders {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	q := req.URL.Query()
	for k, vs := range ctx.Query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()
	resp, err := upstreamClient.Do(req)
	if err != nil {
		setForwardError(ctx, err, "provider request failed")
		return
	}
	ctx.HttpResp = resp
	defer resp.Body.Close()

	if strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		forwardStream(ctx)
	} else {
		forwardBuffered(ctx)
	}
	ctx.MarkLatency(time.Now())
}

// hopByHopHeaders RFC 7230 §6.1 定义的逐跳头（key 为 Canonical 形式）。
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

// copyUpstreamHeaders 复制上游响应头到客户端，剥掉逐跳头及 Connection 点名的头。
func copyUpstreamHeaders(ctx *types.Context) {
	src := ctx.HttpResp.Header
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

// forwardBuffered 读取完整的非流式响应后一次性写回客户端。
func forwardBuffered(ctx *types.Context) {
	body, err := io.ReadAll(ctx.HttpResp.Body)
	if err != nil {
		setForwardError(ctx, err, "provider unstream read failed")
		return
	}
	ctx.RespBody = body
	ctx.ResponseComplete = true

	copyUpstreamHeaders(ctx)
	ctx.Writer.WriteStatusCode(ctx.HttpResp.StatusCode)
	ctx.ResponseStatusCode = ctx.HttpResp.StatusCode
	ctx.ResponseCommitted = true
	if len(body) == 0 {
		return
	}
	if _, err := ctx.Writer.Write(body); err != nil {
		setForwardError(ctx, err, "client unstream response write failed")
	}
}

// forwardStream 透传 SSE 并缓存原始响应，不解析协议事件。
//
// 首个读取块直接写给客户端；后续每次读取到新块后，才写出上一个块，最后一个块
// 等上游返回 EOF 后写出。这样常见的末尾完成事件会随 EOF 一并交付，客户端收到
// 完成事件立即断开时，Forward 通常已不再进行下一次上游读取。
func forwardStream(ctx *types.Context) {
	ctx.Stream = true
	copyUpstreamHeaders(ctx)
	ctx.Writer.WriteStatusCode(ctx.HttpResp.StatusCode)
	ctx.ResponseStatusCode = ctx.HttpResp.StatusCode
	ctx.ResponseCommitted = true

	var (
		buf       bytes.Buffer
		pending   []byte
		firstSent bool
	)
	chunk := make([]byte, 4*1024)
	for {
		n, readErr := ctx.HttpResp.Body.Read(chunk)
		if n > 0 {
			ctx.MarkFirstToken(time.Now())
			current := append([]byte(nil), chunk[:n]...)
			_, _ = buf.Write(current)
			if !firstSent {
				if !writeStreamChunk(ctx, current) {
					ctx.RespBody = buf.Bytes()
					return
				}
				firstSent = true
			} else {
				if len(pending) > 0 && !writeStreamChunk(ctx, pending) {
					ctx.RespBody = buf.Bytes()
					return
				}
				pending = current
			}
		}
		if readErr != nil {
			// Reader 即使返回错误也可能已给出有效字节；最后暂存块同样应尝试
			// 交付客户端，不能因上游连接收尾异常而丢掉已读取的数据。
			if len(pending) > 0 && !writeStreamChunk(ctx, pending) {
				ctx.RespBody = buf.Bytes()
				return
			}
			if readErr == io.EOF {
				ctx.ResponseComplete = true
				ctx.RespBody = buf.Bytes()
				return
			}
			setForwardError(ctx, readErr, "provider stream read failed")
			ctx.RespBody = buf.Bytes()
			return
		}
	}
}

// writeStreamChunk 将一个已缓存的 SSE 块写给客户端。
func writeStreamChunk(ctx *types.Context, chunk []byte) bool {
	if _, err := ctx.Writer.Write(chunk); err != nil {
		setForwardError(ctx, err, "client stream response write failed")
		return false
	}
	return true
}

// setForwardError 记录转发错误，统一按服务端错误处理。
func setForwardError(ctx *types.Context, err error, message string) {
	ctx.Err = log.WithStack(err)
	ctx.ErrorMessage = message
	ctx.Code = types.CodeUnknown
}
