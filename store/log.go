package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// InsertRequestLog 插入请求日志
func (s *Session) InsertRequestLog(log *model.RequestLog) error {
	_, err := s.namedExec(
		`INSERT INTO request_logs (api_key, format, provider, path, status_code, request_headers, request_body, response_body, model, input_tokens, output_tokens, total_tokens, error, latency_ms)
		 VALUES (:api_key, :format, :provider, :path, :status_code, :request_headers, :request_body, :response_body, :model, :input_tokens, :output_tokens, :total_tokens, :error, :latency_ms)`,
		log,
	)
	return err
}
