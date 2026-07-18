package store

import (
	"github.com/jmoiron/sqlx"
	"github.com/lijcoder/aiapi/store/base"
)

// Init 使用外部传入的 *sqlx.DB 初始化全局 Storage 及默认 DB
func Init(db *sqlx.DB) error {
	return base.Init(db)
}

// Close 关闭数据库连接
func Close() error {
	return base.Close()
}

// C 返回一个普通 Session，用于非事务 SQL 调用。
func C() *Session {
	return &Session{Session: base.WithContext()}
}

// T 开启事务，fn 中使用 *Session 的所有 SQL 方法自动在事务内执行。
// fn 返回错误时自动回滚，否则提交。
func T(fn func(*Session) error) error {
	return C().Transaction(func(s *Session) error {
		return fn(s)
	})
}

// Session 是数据库会话，业务 SQL 方法挂载在它上面。
type Session struct {
	*base.Session
}

// Transaction 在 Session 的 context 下开启事务，fn 中所有 Session 方法自动使用事务连接。
// fn 返回错误时自动回滚，否则提交。
func (s *Session) Transaction(fn func(*Session) error) error {
	return s.Session.Transaction(func(bs *base.Session) error {
		return fn(&Session{Session: bs})
	})
}

// RawExec 执行任意 SQL，用于测试建表或初始化数据。
func (s *Session) RawExec(query string, args ...any) (int64, error) {
	res, err := dbFrom(s.Ctx).Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// db 是内部对 base.DB 的别名，业务 SQL 文件复用。
type db = base.DB

// dbFrom 是内部对 base.DBFrom 的别名。
var dbFrom = base.DBFrom
