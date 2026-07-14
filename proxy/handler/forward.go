package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/lijcoder/aiapi/proxy/types"
)

// Forward 转发请求到上游
func Forward(ctx *types.Context) {
	req, err := http.NewRequest(ctx.Method, ctx.URL, io.NopCloser(bytes.NewReader(ctx.Body)))
	if err != nil {
		ctx.Err = err
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
		ctx.Err = err
		return
	}
	ctx.HttpResp = resp
}
