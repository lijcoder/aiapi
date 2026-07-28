package handler

import (
	"fmt"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// AuthModel 校验模型定价配置存在、且当前 API Key 有权访问该模型，设置 ctx.ModelInfo
func AuthModel(ctx *types.Context) {
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
	policy, err := store.C().ModelAccess().GetKeyPolicy(ctx.ApiKeyID)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if policy != store.ModelPolicyWhitelist {
		// all 或未配置时放行
		return
	}
	cnt, err := store.C().ModelAccess().CountApiKeyModelAccess(ctx.ApiKeyID, pvd.ID)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if cnt == 0 {
		ctx.Err = log.WithStack(fmt.Errorf("model not allowed for api key: key_id=%d model_id=%d", ctx.ApiKeyID, pvd.ID))
		// 对外表现为「模型不存在」，不暴露权限细节
		ctx.ErrorMessage = fmt.Sprintf("model %s/%s is not available", ctx.ProviderType, ctx.Model)
		ctx.Code = types.CodeModelNotFound
		return
	}
}
