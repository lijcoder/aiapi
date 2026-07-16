package store

import (
	"errors"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/constant"
)

// newSQLiteStore 创建 SQLite 存储实例（不设置 current）
func newSQLiteStore() (*commonStore, error) {
	dbPath := constant.DBFilePath()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, errors.New("database file not found, please create it first: " + dbPath)
	}

	db, err := sqlx.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &commonStore{db: db}, nil
}
