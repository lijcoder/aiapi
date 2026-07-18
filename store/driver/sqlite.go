package driver

import (
	"errors"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// NewSQLite 按指定路径创建 SQLite 数据库连接
func NewSQLite(path string) (*sqlx.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.New("database file not found, please create it first: " + path)
	}

	db, err := sqlx.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
