package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// 模型访问策略
const (
	ModelPolicyAll       = "all"       // 全量放行
	ModelPolicyWhitelist = "whitelist" // 按 apikey_model_access 白名单
)

// ModelAccess 返回 API Key 模型访问策略相关操作的命名空间。
func (s *Session) ModelAccess() *ModelAccessStore {
	return &ModelAccessStore{s: s}
}

// ModelAccessStore 是 API Key 模型访问策略相关操作的命名空间。
// 操作 api_keys.model_policy 字段与 apikey_model_access 白名单表。
type ModelAccessStore struct {
	s *Session
}

// GetKeyPolicy 查询某 API Key 的模型访问策略值。
// key 不存在时返回 ("", nil)。
func (ms *ModelAccessStore) GetKeyPolicy(apiKeyID int64) (string, error) {
	var k model.ApiKey
	err := ms.s.Query(
		`SELECT model_policy FROM api_keys WHERE id = :id`,
		map[string]any{"id": apiKeyID},
	).Get(&k)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return k.ModelPolicy, nil
}

// CountApiKeyModelAccess 统计某 API Key 的模型白名单中是否包含指定模型。
func (ms *ModelAccessStore) CountApiKeyModelAccess(apiKeyID, modelID int64) (int, error) {
	var cnt int
	err := ms.s.Query(
		`SELECT COUNT(*) FROM apikey_model_access WHERE api_key_id = :id AND model_id = :mid`,
		map[string]any{"id": apiKeyID, "mid": modelID},
	).Get(&cnt)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

// ListApiKeyModelIDs 查询某 API Key 的模型白名单 ID 列表（按 model_id 排序）。
func (ms *ModelAccessStore) ListApiKeyModelIDs(apiKeyID int64) ([]int64, error) {
	var ids []int64
	err := ms.s.Query(
		`SELECT model_id FROM apikey_model_access WHERE api_key_id = :id ORDER BY model_id`,
		map[string]any{"id": apiKeyID},
	).Select(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// UpdateKeyPolicy 更新某 API Key 的模型访问策略。
func (ms *ModelAccessStore) UpdateKeyPolicy(apiKeyID int64, policy string) error {
	_, err := ms.s.Query(
		`UPDATE api_keys SET model_policy = :policy WHERE id = :id`,
		map[string]any{"id": apiKeyID, "policy": policy},
	).Exec()
	return err
}

// InsertApiKeyModelAccess 插入一条 API Key 模型白名单（重复则忽略）。
func (ms *ModelAccessStore) InsertApiKeyModelAccess(apiKeyID, modelID int64) error {
	_, err := ms.s.Query(
		`INSERT OR IGNORE INTO apikey_model_access (api_key_id, model_id) VALUES (:api_key_id, :model_id)`,
		map[string]any{"api_key_id": apiKeyID, "model_id": modelID},
	).Exec()
	return err
}

// DeleteApiKeyModelItems 删除某 API Key 的全部模型白名单（Key 删除时级联清理）。
func (ms *ModelAccessStore) DeleteApiKeyModelItems(apiKeyID int64) error {
	_, err := ms.s.Query(
		`DELETE FROM apikey_model_access WHERE api_key_id = :id`,
		map[string]any{"id": apiKeyID},
	).Exec()
	return err
}
