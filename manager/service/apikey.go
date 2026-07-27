package service

import "github.com/lijcoder/aiapi/store"

// ApiKeyService 封装 API Key 相关业务逻辑（事务编排）。
type ApiKeyService struct{}

// NewApiKeyService 创建 ApiKeyService。
func NewApiKeyService() *ApiKeyService { return &ApiKeyService{} }

// SetModelAccess 设置某 API Key 的模型访问策略（事务内全量替换）。
// policy=all 时清空白名单；policy=whitelist 时按 modelIDs 重写白名单。
func (s *ApiKeyService) SetModelAccess(apiKeyID int64, policy string, modelIDs []int64) error {
	return store.C().T(func(ss *store.Session) error {
		if err := ss.ModelAccess().UpdateKeyPolicy(apiKeyID, policy); err != nil {
			return err
		}
		// 清空旧白名单
		if err := ss.ModelAccess().DeleteApiKeyModelItems(apiKeyID); err != nil {
			return err
		}
		// whitelist 策略下写入新白名单
		if policy == store.ModelPolicyWhitelist {
			for _, mid := range modelIDs {
				if err := ss.ModelAccess().InsertApiKeyModelAccess(apiKeyID, mid); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// GetModelAccess 查询某 API Key 的模型访问策略与白名单列表。
// policy 非 whitelist 时白名单列表为空。
func (s *ApiKeyService) GetModelAccess(apiKeyID int64) (policy string, modelIDs []int64, err error) {
	policy, err = store.C().ModelAccess().GetKeyPolicy(apiKeyID)
	if err != nil {
		return
	}
	if policy != store.ModelPolicyWhitelist {
		return
	}
	modelIDs, err = store.C().ModelAccess().ListApiKeyModelIDs(apiKeyID)
	return
}

// Delete 删除 API Key（事务内同时清理模型白名单）。
func (s *ApiKeyService) Delete(apiKeyID int64) error {
	return store.C().T(func(ss *store.Session) error {
		if err := ss.ModelAccess().DeleteApiKeyModelItems(apiKeyID); err != nil {
			return err
		}
		return ss.ApiKey().DeleteApiKey(apiKeyID)
	})
}
