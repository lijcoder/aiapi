package handler

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/proxy/types"
)

// Log 保存 HTTP 请求日志（用于 AddFinally）
func Log(ctx *types.Context) {
	var model string
	var inTokens, outTokens, totalTokens int
	if ctx.Usage != nil {
		model = ctx.Usage.Model
		inTokens = ctx.Usage.InputTokens
		outTokens = ctx.Usage.OutputTokens
		totalTokens = ctx.Usage.TotalTokens
	}

	errMsg := ""
	if ctx.Err != nil {
		errMsg = ctx.Err.Error()
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

	if err := db.InsertRequestLog(ctx.DB, &db.RequestLog{
		ApiKey:         ctx.ApiKey,
		Format:         ctx.Format,
		Provider:       ctx.ProviderType,
		Method:         ctx.Method,
		Path:           ctx.Path,
		StatusCode:     statusCode,
		RequestHeaders: string(headerJSON),
		RequestBody:    string(ctx.Body),
		ResponseBody:   string(ctx.RespBody),
		Model:          model,
		InputTokens:    inTokens,
		OutputTokens:   outTokens,
		TotalTokens:    totalTokens,
		Error:          errMsg,
		LatencyMs:      latency,
	}); err != nil {
		slog.Warn("insert request log failed", "error", err)
	}
}
