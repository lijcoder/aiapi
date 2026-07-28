package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// Log 返回请求日志相关操作的命名空间。
func (s *Session) Log() *LogStore {
	return &LogStore{s: s}
}

// LogStore 是请求日志相关操作的命名空间。
type LogStore struct {
	s *Session
}

// Insert 插入请求日志
func (ls *LogStore) Insert(log *model.RequestLog) error {
	res, err := ls.s.Query(
		`INSERT INTO request_logs (api_key_id, format, provider, path, status_code, request_headers, request_body, response_body, model, input_tokens, output_tokens, total_tokens, error, latency_ms)
		 VALUES (:api_key_id, :format, :provider, :path, :status_code, :request_headers, :request_body, :response_body, :model, :input_tokens, :output_tokens, :total_tokens, :error, :latency_ms)`,
		log,
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = id
	return nil
}
