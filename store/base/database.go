package base

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// DB 是 sqlx DB 和 Tx 的最小公共接口，便于事务内外复用。
type DB interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

// 编译期校验：*sqlx.DB 和 *sqlx.Tx 都实现 DB
var _ DB = (*sqlx.DB)(nil)
var _ DB = (*sqlx.Tx)(nil)

type ctxKey struct{}

var defaultDB DB

// storage 全局实例，负责底层连接管理与事务生命周期。
var storage *Storage

// Storage 通用数据存储封装，只负责连接管理和事务生命周期。
type Storage struct {
	db *sqlx.DB
}

// BeginTx 开启事务
func (s *Storage) BeginTx() (*sqlx.Tx, error) {
	return s.db.Beginx()
}

// Close 关闭数据库连接
func (s *Storage) Close() error {
	return s.db.Close()
}

// SetDefaultDB 设置默认非事务数据库连接，应在应用启动时调用一次。
func SetDefaultDB(d DB) {
	defaultDB = d
}

// DBFrom 从 ctx 中提取当前事务连接，不存在则返回默认连接。
func DBFrom(ctx context.Context) DB {
	if d, ok := ctx.Value(ctxKey{}).(DB); ok {
		return d
	}
	return defaultDB
}

// WithDBContext 将 DB 连接注入 ctx，供事务内传递使用。
func WithDBContext(ctx context.Context, d DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, d)
}

// Init 使用外部传入的 *sqlx.DB 初始化全局 Storage 及默认 DB
func Init(db *sqlx.DB) error {
	storage = &Storage{db: db}
	SetDefaultDB(db)
	slog.Info("store init success", "driver", "sqlx")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if storage != nil {
		return storage.Close()
	}
	return nil
}
