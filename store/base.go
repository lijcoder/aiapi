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
	return base.WithContext().Transaction(func(bs *base.Session) error {
		return fn(&Session{Session: bs})
	})
}

// Session 是数据库会话，业务 SQL 命名空间通过同包各文件中的方法挂载在它上面。
type Session struct {
	*base.Session
}

// Query 创建一个 QueryBuilder，自动根据当前 Session 是否处于事务选择 DB 或 Tx。
// params 支持 nil、map 或 struct（sqlx.Named 原生支持）。
func (s *Session) Query(query string, params any) *base.QueryBuilder {
	return base.NewQuery(base.DBFrom(s.Ctx), base.DriverName(), query, params)
}
