package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetProvider 按类型查询启用中的 Provider
func (s *Session) GetProvider(providerType string) (*model.Provider, error) {
	var p model.Provider
	err := dbFrom(s.Ctx).Get(&p, `SELECT * FROM providers WHERE type = ? AND enabled = 1`, providerType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
