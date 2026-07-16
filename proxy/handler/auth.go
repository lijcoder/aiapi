package handler

import (
	"errors"
	"net/http"

	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/proxy/types"
)

var (
	errMissingKey   = errors.New("missing api key")
	errInvalidKey   = errors.New("invalid api key")
	errUserDisabled = errors.New("user is disabled")
)

// Auth 校验 API Key
func Auth(ctx *types.Context) {
	ctx.ApiKey = parser.ExtractApiKey(http.Header(ctx.OrigHeaders), ctx.Format)
	if ctx.ApiKey == "" {
		ctx.Err = errMissingKey
		ctx.Code = types.CodeUnauthorized
		return
	}

	_, user, err := store.GetApiKey(ctx.ApiKey)
	if err != nil {
		ctx.Err = errInvalidKey
		ctx.Code = types.CodeUnauthorized
		return
	}

	if user == nil {
		ctx.Err = errUserDisabled
		ctx.Code = types.CodeUnauthorized
		return
	}

	ctx.UserID = user.ID
}
