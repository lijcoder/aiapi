package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetProvider 按类型查询启用中的 Provider
func (s *Session) GetProvider(providerType string) (*model.Provider, error) {
	var p model.Provider
	err := s.namedGet(&p, `SELECT * FROM providers WHERE type = :type AND enabled = 1`, map[string]any{"type": providerType})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
