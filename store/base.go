package store

import (
	"github.com/jmoiron/sqlx"
	"github.com/lijcoder/aiapi/store/base"
)

// 暴露 base 包的 PageContext，供 handler 创建后传给 store.C().SetPage。
// PageReq / PageResult 由 manager/base 定义，store 不重复暴露。
type PageContext = base.PageContext

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

// T 在当前 Session 上切换事务连接，fn 中所有方法自动使用事务。
// fn 返回错误时自动回滚，否则提交并恢复原 tx。
func (s *Session) T(fn func(*Session) error) error {
	return s.Session.Transaction(func(bs *base.Session) error {
		return fn(&Session{Session: bs})
	})
}

// SetPage 覆盖 base.Session.SetPage，返回 *store.Session 以支持链式调用（如 store.C().SetPage(pc).Charge().List(...)）。
func (s *Session) SetPage(pc *base.PageContext) *Session {
	s.Session.SetPage(pc)
	return s
}

// Session 是数据库会话，业务 SQL 命名空间通过同包各文件中的方法挂载在它上面。
type Session struct {
	*base.Session
}

// Query 创建一个 QueryBuilder，自动根据当前 Session 是否处于事务选择 DB 或 Tx。
// params 支持 nil、map 或 struct（sqlx.Named 原生支持）。
// QueryBuilder 从 Session 读取分页参数（PageContext），有则自动拦截 Select 分页。
func (s *Session) Query(query string, params any) *base.QueryBuilder {
	return base.NewQuery(s.DB(), base.DriverName(), s.Page(), query, params)
}
