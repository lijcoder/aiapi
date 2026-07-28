package service

import (
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ModelService 封装模型相关业务逻辑。
type ModelService struct{}

// NewModelService 创建 ModelService。
func NewModelService() *ModelService { return &ModelService{} }

// ListAvailableModels 查询某 API Key 在指定 provider 下可用的模型列表。
// 口径与 proxy 鉴权一致：模型必须配置在该 provider 下，且 Key 为 whitelist 策略时
// 只返回白名单内的模型；all（或未配置）策略返回该 provider 全量模型。
func (s *ModelService) ListAvailableModels(provider string, apiKeyID int64) ([]model.Model, error) {
	policy, modelIDs, err := NewApiKeyService().GetModelAccess(apiKeyID)
	if err != nil {
		return nil, err
	}
	if policy != store.ModelPolicyWhitelist {
		return store.C().Model().ListByProvider(provider)
	}
	if len(modelIDs) == 0 {
		return nil, nil
	}
	models, err := store.C().Model().ListByIDs(modelIDs)
	if err != nil {
		return nil, err
	}
	// 白名单按 model_id 全局配置，列表只保留当前 provider 下的条目
	available := make([]model.Model, 0, len(models))
	for _, m := range models {
		if m.Provider == provider {
			available = append(available, m)
		}
	}
	return available, nil
}
