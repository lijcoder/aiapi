package handler

import (
	"errors"
	"fmt"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

var (
	errMissingKey   = errors.New("missing api key")
	errInvalidKey   = errors.New("invalid api key")
	errUserDisabled = errors.New("user is disabled")
)

// Auth 校验 API Key
func Auth(ctx *types.Context) {
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

	// 校验模型定价配置是否存在
	pvd, err := store.GetModel(ctx.ProviderType, ctx.Model)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if pvd == nil {
		ctx.Err = log.WithStack(fmt.Errorf("model not found: %s/%s", ctx.ProviderType, ctx.Model))
		ctx.ErrorMessage = fmt.Sprintf("model not found: %s/%s", ctx.ProviderType, ctx.Model)
		ctx.Code = types.CodeModelNotFound
		return
	}
	ctx.ModelInfo = pvd
}
