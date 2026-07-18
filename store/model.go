package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// GetModel 按 provider+model 查询模型定价配置
func (s *Session) GetModel(provider, name string) (*model.Model, error) {
	var p model.Model
	err := dbFrom(s.Ctx).Get(&p, `SELECT * FROM models WHERE provider = ? AND model = ?`, provider, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// InsertUsage 插入用量记录
func (s *Session) InsertUsage(usage *model.UsageRecord) error {
	_, err := dbFrom(s.Ctx).NamedExec(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens, cost, unlimited)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens, :cost, :unlimited)`,
		usage,
	)
	return err
}
