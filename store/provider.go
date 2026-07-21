package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// Provider 返回 Provider 相关操作的命名空间。
func (s *Session) Provider() *ProviderStore {
	return &ProviderStore{s: s}
}

// ProviderStore 是 Provider 相关操作的命名空间。
type ProviderStore struct {
	s *Session
}

// Get 按类型查询启用中的 Provider
func (ps *ProviderStore) Get(providerType string) (*model.Provider, error) {
	var p model.Provider
	err := ps.s.Query(
		`SELECT * FROM providers WHERE type = :type AND enabled = 1`,
		map[string]any{"type": providerType},
	).Get(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByType 按类型查询 Provider（不过滤 enabled）
func (ps *ProviderStore) GetByType(providerType string) (*model.Provider, error) {
	var p model.Provider
	err := ps.s.Query(
		`SELECT * FROM providers WHERE type = :type`,
		map[string]any{"type": providerType},
	).Get(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List 列出全部 Provider（按 id 倒序）
func (ps *ProviderStore) List() ([]model.Provider, error) {
	var list []model.Provider
	err := ps.s.Query(
		`SELECT * FROM providers ORDER BY id DESC`,
		nil,
	).Select(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Create 新增 Provider，回填 ID
func (ps *ProviderStore) Create(p *model.Provider) error {
	res, err := ps.s.Query(
		`INSERT INTO providers (type, config, enabled) VALUES (:type, :config, :enabled)`,
		map[string]any{
			"type":    p.Type,
			"config":  p.Config,
			"enabled": p.Enabled,
		},
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

// Update 更新 Provider 的 config 和 enabled（type 不可改）
func (ps *ProviderStore) Update(p *model.Provider) error {
	_, err := ps.s.Query(
		`UPDATE providers SET config = :config, enabled = :enabled, updated_at = datetime('now','localtime')
		 WHERE type = :type`,
		map[string]any{
			"type":    p.Type,
			"config":  p.Config,
			"enabled": p.Enabled,
		},
	).Exec()
	return err
}

// SetEnabled 启用/禁用 Provider
func (ps *ProviderStore) SetEnabled(providerType string, enabled bool) error {
	_, err := ps.s.Query(
		`UPDATE providers SET enabled = :enabled, updated_at = datetime('now','localtime') WHERE type = :type`,
		map[string]any{"type": providerType, "enabled": enabled},
	).Exec()
	return err
}
