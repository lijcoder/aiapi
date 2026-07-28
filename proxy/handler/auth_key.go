package handler

import (
	"errors"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

var (
	errMissingKey   = errors.New("missing api key")
	errInvalidKey   = errors.New("invalid api key")
	errUserDisabled = errors.New("user is disabled")
)

// AuthKey 校验 API Key 与关联用户状态，设置 ctx.UserID / ctx.ApiKeyID
func AuthKey(ctx *types.Context) {
	if ctx.ApiKey == "" {
		ctx.Err = log.WithStack(errMissingKey)
		ctx.ErrorMessage = "missing api key"
		ctx.Code = types.CodeUnauthorized
		return
	}
	key, user, err := service.NewApiKeyService().GetKeyAndUser(ctx.ApiKey)
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
	ctx.ApiKeyID = key.ID
}

