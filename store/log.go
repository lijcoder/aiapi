package store

import (
	"github.com/lijcoder/aiapi/store/model"
)

// InsertRequestLog 插入请求日志
func (s *Session) InsertRequestLog(log *model.RequestLog) error {
	_, err := dbFrom(s.Ctx).NamedExec(
		`INSERT INTO request_logs (api_key, format, provider, path, status_code, request_headers, request_body, response_body, model, input_tokens, output_tokens, total_tokens, error, latency_ms)
		 VALUES (:api_key, :format, :provider, :path, :status_code, :request_headers, :request_body, :response_body, :model, :input_tokens, :output_tokens, :total_tokens, :error, :latency_ms)`,
		log,
	)
	return err
}

// DeductUserBudget 扣减用户余额
func (s *Session) DeductUserBudget(userID int64, cost float64) error {
	_, err := dbFrom(s.Ctx).Exec(`UPDATE users SET budget = budget - ? WHERE id = ?`, cost, userID)
	return err
}

// DeductKeyBudget 扣减 API Key 余额
func (s *Session) DeductKeyBudget(key string, cost float64) error {
	_, err := dbFrom(s.Ctx).Exec(`UPDATE api_keys SET budget = budget - ? WHERE key = ?`, cost, key)
	return err
}
