package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// Usage 返回用量记录相关操作的命名空间。
func (s *Session) Usage() *UsageStore {
	return &UsageStore{s: s}
}

// UsageStore 是用量记录相关操作的命名空间。
type UsageStore struct {
	s *Session
}

// Insert 插入用量记录
func (us *UsageStore) Insert(usage *model.UsageRecord) error {
	res, err := us.s.Query(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens, cost, unlimited)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens, :cost, :unlimited)`,
		usage,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	usage.ID = id
	return nil
}
