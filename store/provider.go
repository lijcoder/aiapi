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
