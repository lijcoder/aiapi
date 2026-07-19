package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// Model 返回模型定价相关操作的命名空间。
func (s *Session) Model() *ModelStore {
	return &ModelStore{s: s}
}

// ModelStore 是模型定价相关操作的命名空间。
type ModelStore struct {
	s *Session
}

// Get 按 provider+model 查询模型定价配置
func (ms *ModelStore) Get(provider, name string) (*model.Model, error) {
	var p model.Model
	err := ms.s.Query(
		`SELECT * FROM models WHERE provider = :provider AND model = :model`,
		map[string]any{"provider": provider, "model": name},
	).Get(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
