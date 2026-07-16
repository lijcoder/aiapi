package handler

import (
	"fmt"

	"github.com/lijcoder/aiapi/proxy/types"
	"github.com/lijcoder/aiapi/store"
)

// LoadConfig 从数据库加载 Provider 配置
func LoadConfig(ctx *types.Context) {
	pvd, ok := store.GetProvider(ctx.ProviderType)
	if !ok {
		ctx.Err = fmt.Errorf("provider not found: %s", ctx.ProviderType)
		ctx.Code = types.CodeNotFound
		return
	}
	cfg, err := pvd.ParseConfig()
	if err != nil {
		ctx.Err = fmt.Errorf("invalid provider config: %s", err.Error())
		ctx.Code = types.CodeNotFound
		return
	}
	ctx.URL = cfg.Domain + "/" + ctx.Path
	ctx.ReqHeaders = cfg.Headers
}
