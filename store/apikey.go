package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lijcoder/aiapi/store/model"
)

// ApiKey 返回 API Key 相关操作的命名空间。
func (s *Session) ApiKey() *ApiKeyStore {
	return &ApiKeyStore{s: s}
}

// ApiKeyStore 是 API Key 相关操作的命名空间。
type ApiKeyStore struct {
	s *Session
}

// Get 按 key 查询 API Key 及关联用户
func (ks *ApiKeyStore) Get(key string) (*model.ApiKey, *model.User, error) {
	var k model.ApiKey
	err := ks.s.Query(
		`SELECT * FROM api_keys WHERE key = :key AND enabled = 1`,
		map[string]any{"key": key},
	).Get(&k)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	u, err := ks.s.User().GetByID(k.UserID)
	if err != nil {
		return &k, nil, err
	}
	if u == nil {
		return &k, nil, nil
	}
	return &k, u, nil
}

// GetByID 按 ID 查询 API Key（不过滤 enabled，调用方自行判断归属与状态）
func (ks *ApiKeyStore) GetByID(id int64) (*model.ApiKey, error) {
	var k model.ApiKey
	err := ks.s.Query(
		`SELECT * FROM api_keys WHERE id = :id`,
		map[string]any{"id": id},
	).Get(&k)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// ListByUser 查询某用户的全部 API Key（按 id 倒序）
func (ks *ApiKeyStore) ListByUser(userID int64) ([]model.ApiKey, error) {
	var keys []model.ApiKey
	err := ks.s.Query(
		`SELECT * FROM api_keys WHERE user_id = :user_id ORDER BY id DESC`,
		map[string]any{"user_id": userID},
	).Select(&keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// ListAll 查询全部 API Key（admin 场景）
func (ks *ApiKeyStore) ListAll() ([]model.ApiKey, error) {
	var keys []model.ApiKey
	err := ks.s.Query(
		`SELECT * FROM api_keys ORDER BY id DESC`,
		nil,
	).Select(&keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Count API Key 总数
func (ks *ApiKeyStore) Count() (int64, error) {
	var n int64
	err := ks.s.Query(`SELECT COUNT(*) FROM api_keys`, nil).Get(&n)
	return n, err
}

// Create 创建 API Key，回填 ID
func (ks *ApiKeyStore) Create(k *model.ApiKey) error {
	res, err := ks.s.Query(
		`INSERT INTO api_keys (user_id, key, name, budget, unlimited, enabled)
		 VALUES (:user_id, :key, :name, :budget, :unlimited, :enabled)`,
		k,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	k.ID = id
	return nil
}

// SetEnabled 启用/禁用 API Key
func (ks *ApiKeyStore) SetEnabled(id int64, enabled bool) error {
	_, err := ks.s.Query(
		`UPDATE api_keys SET enabled = :enabled WHERE id = :id`,
		map[string]any{"id": id, "enabled": enabled},
	).Exec()
	return err
}

// UpdateName 修改 API Key 名称
func (ks *ApiKeyStore) UpdateName(id int64, name string) error {
	_, err := ks.s.Query(
		`UPDATE api_keys SET name = :name WHERE id = :id`,
		map[string]any{"id": id, "name": name},
	).Exec()
	return err
}

// UpdateBudget 修改 API Key 的额度与限额模式（unlimited=false 时 budget 生效）
func (ks *ApiKeyStore) UpdateBudget(id int64, budget float64, unlimited bool) error {
	_, err := ks.s.Query(
		`UPDATE api_keys SET budget = :budget, unlimited = :unlimited WHERE id = :id`,
		map[string]any{"id": id, "budget": budget, "unlimited": unlimited},
	).Exec()
	return err
}

// SumLimitedBudgetByUser 统计某用户所有“有限额” API Key 的 budget 之和。
// 用于在修改/创建 key 时校验总和不超过用户余额。
func (ks *ApiKeyStore) SumLimitedBudgetByUser(userID int64) (float64, error) {
	var sum float64
	err := ks.s.Query(
		`SELECT COALESCE(SUM(budget), 0) FROM api_keys WHERE user_id = :user_id AND unlimited = 0`,
		map[string]any{"user_id": userID},
	).Get(&sum)
	if err != nil {
		return 0, err
	}
	return sum, nil
}

// DeleteApiKey 删除 API Key。
func (ks *ApiKeyStore) DeleteApiKey(id int64) error {
	_, err := ks.s.Query(
		`DELETE FROM api_keys WHERE id = :id`,
		map[string]any{"id": id},
	).Exec()
	return err
}

// IsUniqueConstraintErr 判断是否为唯一约束冲突错误。
// SQLite UNIQUE 冲突的扩展错误码为 2067，消息含 "UNIQUE constraint failed"；
// 跨驱动兼容这里同时做字符串匹配。
func IsUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}
