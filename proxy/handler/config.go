package handler

import (
	"errors"

	"github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// LoadConfig 从数据库加载 Provider 配置
func LoadConfig(ctx *types.Context) {
	pvd, err := store.C().GetProvider(ctx.ProviderType)
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
	cfg, err := pvd.ParseConfig()
	if err != nil {
		ctx.Err = log.WithStack(err)
		ctx.ErrorMessage = "invalid provider config"
		ctx.Code = types.CodeNotFound
		return
	}
	ctx.URL = cfg.Domain + "/" + ctx.Path
	ctx.ReqHeaders = cfg.Headers
}
