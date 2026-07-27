package handler

import (
	"bytes"
	"net/http"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Forward 转发请求到上游
func Forward(ctx *types.Context) {
	// 注意：body 必须直接传 *bytes.Reader，不能用 io.NopCloser 包装。
	// NewRequest 内部对 *bytes.Reader 做类型断言以设置 ContentLength 和 GetBody；
	// 包装后断言失败 → ContentLength=0 → 走 chunked 编码，且连接无法复用。
	req, err := http.NewRequest(ctx.Method, ctx.URL, bytes.NewReader(ctx.Body))
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "upstream request failed"
		return
	}
	ctx.HttpResp = resp
}
