package handler

import (
	"errors"

	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/proxy/types"
)

var (
	errMissingKey   = errors.New("missing api key")
	errInvalidKey   = errors.New("invalid api key")
	errUserDisabled = errors.New("user is disabled")
)

// Auth 校验 API Key
func Auth(ctx *types.Context) {
	if ctx.ApiKey == "" {
		ctx.Err = errMissingKey
		ctx.Code = types.CodeUnauthorized
		return
	}

	_, user, err := db.LookupApiKey(ctx.DB, ctx.ApiKey)
	if err != nil {
		ctx.Err = errInvalidKey
		ctx.Code = types.CodeUnauthorized
		return
	}

	// key 有效但关联用户异常（不存在或禁用）
	if user == nil {
		ctx.Err = errUserDisabled
		ctx.Code = types.CodeUnauthorized
		return
	}

	ctx.UserID = user.ID
}
