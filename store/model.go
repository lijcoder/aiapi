package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetModel 按 provider+model 查询模型定价配置
func (s *Session) GetModel(provider, name string) (*model.Model, error) {
	var p model.Model
	err := s.namedGet(&p, `SELECT * FROM models WHERE provider = :provider AND model = :model`, map[string]any{"provider": provider, "model": name})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
