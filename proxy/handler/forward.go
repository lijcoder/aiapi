package handler

import (
	"bytes"
	"net"
	"net/http"
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

// Forward 转发请求到上游
func Forward(ctx *types.Context) {
	// 注意：body 必须直接传 *bytes.Reader，不能用 io.NopCloser 包装。
	// NewRequest 内部对 *bytes.Reader 做类型断言以设置 ContentLength 和 GetBody；
	// 包装后断言失败 → ContentLength=0 → 走 chunked 编码，且连接无法复用。
	// 携带请求 context：客户端断开时取消上游请求，及时释放连接与 goroutine；
	// 流式阶段 body.Read 也会被取消，透传循环随之退出。
	req, err := http.NewRequestWithContext(ctx.Ctx, ctx.Method, ctx.URL, bytes.NewReader(ctx.Body))
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
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
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "upstream request failed"
		return
	}
	ctx.HttpResp = resp
}
