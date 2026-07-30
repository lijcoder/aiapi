package service

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ApiKeyHash 计算 API Key 的存储哈希（SHA-256 hex）。
// 鉴权、创建、迁移统一用本函数，保证哈希口径一致。
func ApiKeyHash(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// EncryptApiKey 用 AES-256-GCM 加密 API Key 原文，输出 base64(nonce+ciphertext)。
// 用于 api_keys.key_enc 落库，查看接口解密还原明文。
func EncryptApiKey(plain string) (string, error) {
	return encryptWithPurpose(plain, purposeAPIKey)
}

// DecryptApiKey 解密 api_keys.key_enc 还原明文 key。
func DecryptApiKey(enc string) (string, error) {
	return decryptWithPurpose(enc, purposeAPIKey)
}

// key 展示片段的截取长度：前缀 sk- + 3 位 hex，后缀 3 位 hex。
const (
	apiKeyPrefixLen = 6 // sk- + 3 位 hex
	apiKeySuffixLen = 3 // 末尾 3 位 hex
)

// ApiKeyShow 从明文 key 生成展示串（形态 sk-abc****xyz），落库存储。
// key 原文不落库，此展示串是用户辨认 key 的唯一线索，创建与迁移统一口径。
// 过短的 key 防御性处理：直接返回原文（本就不会发生，key 为 sk-+64hex）。
func ApiKeyShow(key string) string {
	if len(key) <= apiKeyPrefixLen {
		return key
	}
	show := key[:apiKeyPrefixLen] + "****"
	if len(key) > apiKeyPrefixLen+apiKeySuffixLen {
		show += key[len(key)-apiKeySuffixLen:]
	}
	return show
}

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

// GetModelAccessDetail 查询某 API Key 的模型访问策略与白名单模型详情。
// 供管理端展示「已选模型」：白名单 ID 换为模型信息，policy 非 whitelist 或无白名单时返回空切片。
func (s *ApiKeyService) GetModelAccessDetail(apiKeyID int64) (policy string, models []model.Model, err error) {
	policy, modelIDs, err := s.GetModelAccess(apiKeyID)
	if err != nil {
		return
	}
	models = []model.Model{}
	if len(modelIDs) == 0 {
		return
	}
	models, err = store.C().Model().ListByIDs(modelIDs)
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

// GetKeyAndUser 按 API Key 查询 key 及其关联用户（两步查询）。
// key 原文不落库，先算哈希再比对；key 不存在或未启用、用户不存在时返回 nil。
func (s *ApiKeyService) GetKeyAndUser(apiKey string) (*model.ApiKey, *model.User, error) {
	k, err := store.C().ApiKey().GetByKeyHash(ApiKeyHash(apiKey))
	if err != nil {
		return nil, nil, err
	}
	if k == nil {
		return nil, nil, nil
	}
	u, err := store.C().User().GetByID(k.UserID)
	if err != nil {
		return k, nil, err
	}
	return k, u, nil
}
