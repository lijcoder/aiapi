package handler

import (
	"errors"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/service"
)

// LoadConfig 从数据库加载 Provider 配置（service 统一入口：查库+解密+解析一次完成）
func LoadConfig(ctx *types.Context) {
	pvd, cfg, err := service.GetProviderConfig(ctx.ProviderType)
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = types.InternalServerError
		ctx.Code = types.CodeUnknown
		return
	}
	if pvd == nil {
		ctx.Err = log.WithStack(errors.New("provider not found"))
		ctx.ErrorMessage = "provider not found: " + ctx.ProviderType
		ctx.Code = types.CodeNotFound
		return
	}
	if cfg == nil {
		ctx.Err = log.WithStack(errors.New("invalid provider config: " + ctx.ProviderType))
		ctx.ErrorMessage = "invalid provider config"
		ctx.Code = types.CodeNotFound
		return
	}
	ctx.URL = cfg.Domain + "/" + ctx.Path
	ctx.ReqHeaders = cfg.Headers
}
