package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// Log 保存 HTTP 请求日志（用于 AddFinally）
func Log(ctx *types.Context) {
	var inTokens, outTokens, totalTokens int
	if ctx.Usage != nil {
		inTokens = ctx.Usage.InputTokens
		outTokens = ctx.Usage.OutputTokens
		totalTokens = ctx.Usage.TotalTokens
	}
	errMsg := ""
	if ctx.Err != nil {
		errMsg = ctx.ErrorMessage
	}
	headers := make(map[string][]string)
	for k, vs := range ctx.OrigHeaders {
		low := strings.ToLower(k)
		if low == "authorization" || low == "cookie" || low == "set-cookie" || low == "x-api-key" || low == "x-goog-api-key" {
			headers[k] = []string{"***"}
			continue
		}
		headers[k] = vs
	}
	headerJSON, _ := json.Marshal(headers)
	statusCode := 0
	if ctx.HttpResp != nil {
		statusCode = ctx.HttpResp.StatusCode
	}
	latency := time.Since(ctx.StartTime).Milliseconds()
	if err := store.C().Log().Insert(&model.RequestLog{
		ApiKey:         ctx.ApiKey,
		Format:         ctx.Format,
		Provider:       ctx.ProviderType,
		Path:           ctx.Path,
		StatusCode:     statusCode,
		RequestHeaders: string(headerJSON),
		RequestBody:    string(ctx.Body),
		ResponseBody:   string(ctx.RespBody),
		Model:          ctx.Model,
		InputTokens:    inTokens,
		OutputTokens:   outTokens,
		TotalTokens:    totalTokens,
		Error:          errMsg,
		LatencyMs:      latency,
	}); err != nil {
		ctx.OtherErrs = append(ctx.OtherErrs, log.WithStack(err))
	}
}
