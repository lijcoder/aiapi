package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/lijcoder/aiapi/constant"
	_ "github.com/mattn/go-sqlite3"
)

// Init 初始化数据库：打开连接
func Init() (*sql.DB, error) {
	dbPath := constant.DBFilePath()

	// 数据库文件必须手动创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, errors.New("database file not found, please create it first: " + dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	slog.Info("db init success", "path", dbPath)
	return db, nil
}

// ==================== Provider ====================

// GetProvider 查询指定 type 的 provider
func GetProvider(db *sql.DB, providerType string) (*Provider, bool) {
	var p Provider
	err := db.QueryRow(
		`SELECT id, type, config, enabled, created_at, updated_at FROM providers WHERE type = ? AND enabled = 1`,
		providerType,
	).Scan(&p.ID, &p.Type, &p.Config, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, false
	}
	return &p, true
}

// ListProviders 查询所有启用的 provider
func ListProviders(db *sql.DB) ([]Provider, error) {
	rows, err := db.Query(`SELECT id, type, config, enabled, created_at, updated_at FROM providers WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Type, &p.Config, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// AddProvider 新增 provider
func AddProvider(db *sql.DB, p *Provider) error {
	config := p.Config
	if config == "" {
		config = "{}"
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO providers (type, config, enabled, updated_at) VALUES (?, ?, ?, datetime('now'))`,
		p.Type, config, boolToInt(p.Enabled),
	)
	return err
}

// DeleteProvider 删除 provider
func DeleteProvider(db *sql.DB, providerType string) error {
	_, err := db.Exec(`DELETE FROM providers WHERE type = ?`, providerType)
	return err
}

// ParseConfig 解析 provider.config JSON
func (p *Provider) ParseConfig() (*ProviderConfig, error) {
	var cfg ProviderConfig
	if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ==================== User ====================

func CreateUser(db *sql.DB, user *User) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO users (name, budget, enabled) VALUES (?, ?, ?)`,
		user.Name, user.Budget, boolToInt(user.Enabled),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ==================== ApiKey ====================

func CreateApiKey(db *sql.DB, key *ApiKey) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO api_keys (user_id, key, name, enabled) VALUES (?, ?, ?, ?)`,
		key.UserID, key.Key, key.Name, boolToInt(key.Enabled),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// LookupApiKey 根据 key 查找对应的 ApiKey 和关联用户
func LookupApiKey(db *sql.DB, apikey string) (*ApiKey, *User, error) {
	var k ApiKey
	err := db.QueryRow(
		`SELECT id, user_id, key, name, enabled FROM api_keys WHERE key = ? AND enabled = 1`,
		apikey,
	).Scan(&k.ID, &k.UserID, &k.Key, &k.Name, &k.Enabled)
	if err != nil {
		return nil, nil, err
	}

	var u User
	err = db.QueryRow(
		`SELECT id, name, budget, enabled, created_at FROM users WHERE id = ? AND enabled = 1`,
		k.UserID,
	).Scan(&u.ID, &u.Name, &u.Budget, &u.Enabled, &u.CreatedAt)
	if err != nil {
		return &k, nil, err
	}
	return &k, &u, nil
}

// ==================== UsageRecord ====================

func InsertUsage(db *sql.DB, usage *UsageRecord) error {
	_, err := db.Exec(
		`INSERT INTO usage_records (user_id, api_key, provider, model, input_tokens, output_tokens, total_tokens, request_id, stream)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usage.UserID, usage.ApiKey, usage.Provider, usage.Model,
		usage.InputTokens, usage.OutputTokens, usage.TotalTokens,
		usage.RequestID, boolToInt(usage.Stream),
	)
	return err
}

// RecordUsageAsync 异步记录用量
func RecordUsageAsync(db *sql.DB, usage *UsageRecord) {
	go func() {
		if err := InsertUsage(db, usage); err != nil {
			slog.Error("record usage failed", "err", err)
		}
	}()
}

// ==================== Budget ====================

func GetUserBudget(db *sql.DB, userID int64) (float64, error) {
	var budget float64
	err := db.QueryRow(`SELECT budget FROM users WHERE id = ?`, userID).Scan(&budget)
	return budget, err
}

func GetUserTotalCost(db *sql.DB, userID int64) (float64, error) {
	var total sql.NullFloat64
	err := db.QueryRow(
		`SELECT SUM(cost) FROM usage_records WHERE user_id = ?`,
		userID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func GetRemainingBudget(db *sql.DB, userID int64) (float64, error) {
	budget, err := GetUserBudget(db, userID)
	if err != nil {
		return 0, err
	}
	if budget <= 0 {
		return -1, nil // 无限制
	}
	total, err := GetUserTotalCost(db, userID)
	if err != nil {
		return 0, err
	}
	remaining := budget - total
	return remaining, nil
}

// ==================== Utils ====================

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
