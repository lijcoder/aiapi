package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetApiKey 按 key 查询 API Key 及关联用户
func (s *Session) GetApiKey(key string) (*model.ApiKey, *model.User, error) {
	var k model.ApiKey
	err := s.namedGet(&k, `SELECT * FROM api_keys WHERE key = :key AND enabled = 1`, map[string]any{"key": key})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	u, err := s.GetUserByID(k.UserID)
	if err != nil {
		return &k, nil, err
	}
	if u == nil {
		return &k, nil, nil
	}
	return &k, u, nil
}
