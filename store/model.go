package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lijcoder/aiapi/store/model"
)

// Model 返回模型相关操作的命名空间。
func (s *Session) Model() *ModelStore {
	return &ModelStore{s: s}
}

// ModelStore 是模型相关操作的命名空间。
type ModelStore struct {
	s *Session
}

// Get 按 provider+model 查询模型配置
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

// List 查询全部模型配置，支持按 provider/model 模糊搜索
func (ms *ModelStore) List(providerKw, modelKw string) ([]model.Model, error) {
	q := `SELECT * FROM models`
	args := map[string]any{}
	var conds []string
	if providerKw != "" {
		conds = append(conds, "provider LIKE :provider_kw")
		args["provider_kw"] = "%" + providerKw + "%"
	}
	if modelKw != "" {
		conds = append(conds, "model LIKE :model_kw")
		args["model_kw"] = "%" + modelKw + "%"
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY id DESC"
	var models []model.Model
	err := ms.s.Query(q, args).Select(&models)
	if err != nil {
		return nil, err
	}
	return models, nil
}

// GetByID 按 ID 查询模型
func (ms *ModelStore) GetByID(id int64) (*model.Model, error) {
	var m model.Model
	err := ms.s.Query(
		`SELECT * FROM models WHERE id = :id`,
		map[string]any{"id": id},
	).Get(&m)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 新增模型，回填 ID
func (ms *ModelStore) Create(m *model.Model) error {
	res, err := ms.s.Query(
		`INSERT INTO models (provider, model, input_cache_hit_price, input_cache_miss_price, output_price, max_context_tokens, max_completion_tokens, supports_text, supports_image, supports_video)
		 VALUES (:provider, :model, :input_cache_hit_price, :input_cache_miss_price, :output_price, :max_context_tokens, :max_completion_tokens, :supports_text, :supports_image, :supports_video)`,
		map[string]any{
			"provider":               m.Provider,
			"model":                  m.Model,
			"input_cache_hit_price":  m.InputCacheHitPrice,
			"input_cache_miss_price": m.InputCacheMissPrice,
			"output_price":           m.OutputPrice,
			"max_context_tokens":     m.MaxContextTokens,
			"max_completion_tokens":  m.MaxCompletionTokens,
			"supports_text":          m.SupportsText,
			"supports_image":         m.SupportsImage,
			"supports_video":         m.SupportsVideo,
		},
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

// Update 更新模型（provider+model 不可改）
func (ms *ModelStore) Update(m *model.Model) error {
	_, err := ms.s.Query(
		`UPDATE models SET input_cache_hit_price=:input_cache_hit_price, input_cache_miss_price=:input_cache_miss_price,
		 output_price=:output_price, max_context_tokens=:max_context_tokens, max_completion_tokens=:max_completion_tokens,
		 supports_text=:supports_text, supports_image=:supports_image, supports_video=:supports_video
		 WHERE id=:id`,
		map[string]any{
			"id":                     m.ID,
			"input_cache_hit_price":  m.InputCacheHitPrice,
			"input_cache_miss_price": m.InputCacheMissPrice,
			"output_price":           m.OutputPrice,
			"max_context_tokens":     m.MaxContextTokens,
			"max_completion_tokens":  m.MaxCompletionTokens,
			"supports_text":          m.SupportsText,
			"supports_image":         m.SupportsImage,
			"supports_video":         m.SupportsVideo,
		},
	).Exec()
	return err
}

// Delete 删除模型
func (ms *ModelStore) Delete(id int64) error {
	_, err := ms.s.Query(
		`DELETE FROM models WHERE id = :id`,
		map[string]any{"id": id},
	).Exec()
	return err
}
