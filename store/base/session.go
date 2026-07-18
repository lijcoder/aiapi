package base

import (
	"context"
)

// Session 表示一个已绑定 context 的数据库会话
type Session struct {
	Ctx context.Context
}

// WithContext 返回一个使用默认 context.Background() 的 Session，用于后续 SQL 方法调用。
func WithContext() *Session {
	return &Session{Ctx: context.Background()}
}

// Transaction 在 Session 的 context 下开启事务，fn 中所有 Session 方法自动使用事务连接。
// fn 返回错误时自动回滚，否则提交。
func (s *Session) Transaction(fn func(*Session) error) error {
	tx, err := storage.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txCtx := WithDBContext(s.Ctx, tx)
	if err := fn(&Session{Ctx: txCtx}); err != nil {
		return err
	}

	return tx.Commit()
}
