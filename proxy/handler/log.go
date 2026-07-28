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
		if isSensitiveHeader(k) {
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
		ApiKeyID:       ctx.ApiKeyID,
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

// sensitiveHeaderKeys 请求头脱敏关键词（小写子串匹配）。
// 用「包含匹配」而非精确名单：各家上游/网关的凭证头命名五花八门
// （x-api-key、api-key、x-auth-token、x-goog-api-key、proxy-authorization...），
// 名单制必有漏网之鱼。宁可误伤普通头，不可漏脱凭证头。
var sensitiveHeaderKeys = []string{"auth", "cookie", "key", "token", "secret", "password"}

// isSensitiveHeader 判断请求头是否可能携带凭证（需脱敏）。
func isSensitiveHeader(name string) bool {
	low := strings.ToLower(name)
	for _, kw := range sensitiveHeaderKeys {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}
