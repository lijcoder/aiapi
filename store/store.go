package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// Store 数据存储接口，可替换实现（SQLite / MySQL / ...）
type Store interface {
	GetProvider(providerType string) (*Provider, error)
	GetApiKey(key string) (*ApiKey, *User, error)
	InsertUsage(usage *UsageRecord) error
	InsertRequestLog(log *RequestLog) error
	Close() error
}

var current Store

// 包级包装函数 — handler 调用 store.GetProvider() 走 current 转发
func GetProvider(t string) (*Provider, error)        { return current.GetProvider(t) }
func GetApiKey(k string) (*ApiKey, *User, error) { return current.GetApiKey(k) }
func InsertUsage(u *UsageRecord) error               { return current.InsertUsage(u) }
func InsertRequestLog(l *RequestLog) error           { return current.InsertRequestLog(l) }
func Close() error                                    { return current.Close() }

// commonStore 公共存储实现，所有驱动共用同一套 SQL
type commonStore struct {
	db *sqlx.DB
}

func (s *commonStore) GetProvider(providerType string) (*Provider, error) {
	var p Provider
	err := s.db.Get(&p, `SELECT * FROM providers WHERE type = ? AND enabled = 1`, providerType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *commonStore) GetApiKey(key string) (*ApiKey, *User, error) {
	var k ApiKey
	err := s.db.Get(&k, `SELECT * FROM api_keys WHERE key = ? AND enabled = 1`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	var u User
	err = s.db.Get(&u, `SELECT * FROM users WHERE id = ? AND enabled = 1`, k.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return &k, nil, nil
	}
	if err != nil {
		return &k, nil, err
	}
	return &k, &u, nil
}

func (s *commonStore) InsertUsage(usage *UsageRecord) error {
	_, err := s.db.NamedExec(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens)`,
		usage,
	)
	return err
}

func (s *commonStore) InsertRequestLog(log *RequestLog) error {
	_, err := s.db.NamedExec(
		`INSERT INTO request_logs (api_key, format, provider, path, status_code, request_headers, request_body, response_body, model, input_tokens, output_tokens, total_tokens, error, latency_ms)
		 VALUES (:api_key, :format, :provider, :path, :status_code, :request_headers, :request_body, :response_body, :model, :input_tokens, :output_tokens, :total_tokens, :error, :latency_ms)`,
		log,
	)
	return err
}

func (s *commonStore) Close() error {
	return s.db.Close()
}

// Init 初始化存储（当前为 SQLite）
func Init() error {
	s, err := newSQLiteStore()
	if err != nil {
		return err
	}
	current = s
	slog.Info("store init success", "driver", "sqlite")
	return nil
}

// ParseConfig 解析 provider.config JSON
func (p *Provider) ParseConfig() (*ProviderConfig, error) {
	var cfg ProviderConfig
	if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
