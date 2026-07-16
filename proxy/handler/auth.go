package handler

import (
	"errors"
	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
	"net/http"
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
		ctx.Err = log.WithStack(errMissingKey)
		ctx.ErrorMessage = "missing api key"
		ctx.Code = types.CodeUnauthorized
		return
	}
	key, user, err := store.GetApiKey(ctx.ApiKey)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if key == nil {
		ctx.Err = log.WithStack(errInvalidKey)
		ctx.ErrorMessage = "invalid api key"
		ctx.Code = types.CodeUnauthorized
		return
	}
	if user == nil {
		ctx.Err = log.WithStack(errUserDisabled)
		ctx.ErrorMessage = "user is disabled"
		ctx.Code = types.CodeUnauthorized
		return
	}
	ctx.UserID = user.ID
}
