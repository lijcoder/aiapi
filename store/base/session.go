package base

import "github.com/jmoiron/sqlx"

// Session 是数据库会话，保存事务连接与分页状态。
// 每次查询经 store.C() 新建，请求级临时对象，不共享。
type Session struct {
	tx   *sqlx.Tx       // 事务连接，nil 则用默认 DB
	page *PageContext   // 分页参数，nil 则不分页
}

// WithContext 返回一个空 Session（非事务、无分页）。
func WithContext() *Session {
	return &Session{}
}

// Transaction 在当前 Session 上开启事务，fn 中所有方法自动使用事务连接。
// 嵌套调用（事务内再调 T）复用当前 tx，不开新事务，保证原子性。
// 顶层调用：fn 返回错误回滚，否则提交并清空 tx。
func (s *Session) Transaction(fn func(*Session) error) error {
	// 已在事务内：复用当前 tx，不开新事务（嵌套）
	if s.tx != nil {
		return fn(s)
	}
	// 顶层事务：开新事务
	tx, err := storage.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	s.tx = tx
	err = fn(s)
	s.tx = nil
	if err != nil {
		return err
	}
	return tx.Commit()
}

// SetPage 设置分页参数，拦截器对后续 Select 自动分页，total 写回 pc。返回 s 支持链式调用。
func (s *Session) SetPage(pc *PageContext) *Session {
	s.page = pc
	return s
}

// ClearPage 清除分页参数，后续 Select 不再分页。
func (s *Session) ClearPage() {
	s.page = nil
}

// DB 返回当前执行器：事务中返回 tx，否则返回默认 DB。
func (s *Session) DB() sqlx.Ext {
	if s.tx != nil {
		return s.tx
	}
	return defaultDB
}

// Page 返回当前分页参数，nil 表示不分页。
func (s *Session) Page() *PageContext {
	return s.page
}
