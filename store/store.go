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
	GetModel(provider, model string) (*Model, error)
	InsertUsage(usage *UsageRecord) error
	InsertRequestLog(log *RequestLog) error
	DeductUserBudget(userID int64, cost float64) error
	DeductKeyBudget(key string, cost float64) error
	Recharge(userID int64, amount float64, operator, remark string) (*RechargeRecord, error)
	Close() error
}

var current Store

// 包级包装函数 — handler 调用 store.GetProvider() 走 current 转发
func GetProvider(t string) (*Provider, error)        { return current.GetProvider(t) }
func GetApiKey(k string) (*ApiKey, *User, error) { return current.GetApiKey(k) }
func GetModel(provider, model string) (*Model, error) { return current.GetModel(provider, model) }
func InsertUsage(u *UsageRecord) error               { return current.InsertUsage(u) }
func InsertRequestLog(l *RequestLog) error           { return current.InsertRequestLog(l) }
func DeductUserBudget(id int64, cost float64) error  { return current.DeductUserBudget(id, cost) }
func DeductKeyBudget(key string, cost float64) error { return current.DeductKeyBudget(key, cost) }
func Recharge(userID int64, amount float64, operator, remark string) (*RechargeRecord, error) {
	return current.Recharge(userID, amount, operator, remark)
}
func Close() error { return current.Close() }

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

func (s *commonStore) GetModel(provider, model string) (*Model, error) {
	var p Model
	err := s.db.Get(&p, `SELECT * FROM models WHERE provider = ? AND model = ?`, provider, model)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *commonStore) InsertUsage(usage *UsageRecord) error {
	_, err := s.db.NamedExec(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream, cached_tokens, reasoning_tokens, cost, unlimited)
		 VALUES (:user_id, :api_key, :provider, :model, :input_tokens, :output_tokens, :total_tokens, :request_id, :stream, :cached_tokens, :reasoning_tokens, :cost, :unlimited)`,
		usage,
	)
	return err
}

func (s *commonStore) DeductUserBudget(userID int64, cost float64) error {
	_, err := s.db.Exec(`UPDATE users SET budget = budget - ? WHERE id = ?`, cost, userID)
	return err
}

func (s *commonStore) DeductKeyBudget(key string, cost float64) error {
	_, err := s.db.Exec(`UPDATE api_keys SET budget = budget - ? WHERE key = ?`, cost, key)
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

func (s *commonStore) Recharge(userID int64, amount float64, operator, remark string) (*RechargeRecord, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var before float64
	err = tx.Get(&before, `SELECT budget FROM users WHERE id = ? AND enabled = 1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	after := before + amount
	_, err = tx.Exec(`UPDATE users SET budget = ? WHERE id = ?`, after, userID)
	if err != nil {
		return nil, err
	}

	rec := &RechargeRecord{
		UserID:        userID,
		Amount:        amount,
		BalanceBefore: before,
		BalanceAfter:  after,
		Operator:      operator,
		Remark:        remark,
	}
	res, err := tx.NamedExec(
		`INSERT INTO recharge_records (user_id, amount, balance_before, balance_after, operator, remark)
		 VALUES (:user_id, :amount, :balance_before, :balance_after, :operator, :remark)`,
		rec,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()
	rec.ID = id
	return rec, nil
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
