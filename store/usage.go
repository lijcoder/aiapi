package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// InsertUsage 插入用量记录
func (s *Session) InsertUsage(usage *model.UsageRecord) error {
	_, err := s.namedExec(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens, cost, unlimited)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens, :cost, :unlimited)`,
		usage,
	)
	return err
}
