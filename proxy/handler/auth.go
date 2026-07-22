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
	key, user, err := store.C().ApiKey().Get(ctx.ApiKey)
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

	// 校验模型定价配置是否存在
	pvd, err := store.C().Model().Get(ctx.ProviderType, ctx.Model)
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

	// 校验该 API Key 是否有权访问此模型
	// 策略为 all（或未配置）时放行；为 whitelist 时按 apikey_model_access 白名单判定
	allowed, err := store.C().Model().IsModelAllowedByApiKey(ctx.ApiKeyID, pvd.ID)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if !allowed {
		ctx.Err = log.WithStack(fmt.Errorf("model not allowed for api key: key_id=%d model_id=%d", ctx.ApiKeyID, pvd.ID))
		// 对外表现为「模型不存在」，不暴露权限细节
		ctx.ErrorMessage = fmt.Sprintf("model %s/%s is not available", ctx.ProviderType, ctx.Model)
		ctx.Code = types.CodeModelNotFound
		return
	}
}
