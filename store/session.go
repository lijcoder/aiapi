package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lijcoder/aiapi/store/model"
)

// UserSession 返回登录会话相关操作的命名空间。
// 方法名用 UserSession 以避免与 store.Session 冲突。
func (s *Session) UserSession() *UserSessionStore {
	return &UserSessionStore{s: s}
}

// UserSessionStore 是登录会话相关操作的命名空间。
type UserSessionStore struct {
	s *Session
}

// Create 创建一条登录会话
func (ss *UserSessionStore) Create(token string, userID int64, expiresAt time.Time) error {
	_, err := ss.s.Query(
		`INSERT INTO user_sessions (token, user_id, expires_at)
		 VALUES (:token, :user_id, :expires_at)`,
		map[string]any{
			"token":      token,
			"user_id":    userID,
			"expires_at": expiresAt,
		},
	).Exec()
	return err
}

// GetByToken 按 token 查询会话
func (ss *UserSessionStore) GetByToken(token string) (*model.UserSession, error) {
	var us model.UserSession
	err := ss.s.Query(
		`SELECT * FROM user_sessions WHERE token = :token`,
		map[string]any{"token": token},
	).Get(&us)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &us, nil
}

// Delete 删除一条会话（登出）
func (ss *UserSessionStore) Delete(token string) error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions WHERE token = :token`,
		map[string]any{"token": token},
	).Exec()
	return err
}

// DeleteExpired 清理已过期会话
func (ss *UserSessionStore) DeleteExpired() error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions WHERE expires_at < datetime('now','localtime')`,
		nil,
	).Exec()
	return err
}
