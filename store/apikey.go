package store

import (
	"database/sql"
	"errors"

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
