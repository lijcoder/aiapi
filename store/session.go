package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lijcoder/aiapi/store/base"
	"github.com/lijcoder/aiapi/store/model"
)

// UserSession 返回登录会话相关操作的命名空间。
// 方法名用 UserSession 以避免与 store.Session 冲突。
func (s *Session) UserSession() *UserSessionStore {
	return &UserSessionStore{s: s}
}

// UserSessionStore 是登录会话相关操作的命名空间。
// 存储的是 refresh token（哈希后），access JWT 不落库。
type UserSessionStore struct {
	s *Session
}

// Create 创建一条登录会话（refresh token）。
//   - token：refresh token 的哈希值（调用方负责哈希）
//   - familyID：登录链标识，重用检测用
//   - expiresAt：滑动过期时间
//   - absoluteExpiresAt：绝对过期上限（不随刷新顺延）
//   - ua/ip：User-Agent 摘要与登录 IP，仅展示用
func (ss *UserSessionStore) Create(token, familyID string, userID int64, expiresAt, absoluteExpiresAt time.Time, ua, ip string) error {
	_, err := ss.s.Query(
		`INSERT INTO user_sessions (token, family_id, user_id, expires_at, absolute_expires_at, ua, ip)
		 VALUES (:token, :family_id, :user_id, :expires_at, :absolute_expires_at, :ua, :ip)`,
		map[string]any{
			"token":               token,
			"family_id":           familyID,
			"user_id":             userID,
			"expires_at":          expiresAt,
			"absolute_expires_at": absoluteExpiresAt,
			"ua":                  ua,
			"ip":                  ip,
		},
	).Exec()
	return err
}

// GetByToken 按 token（哈希值）查询会话。
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

// GetActiveByFamily 查询某 family 下当前活跃（未删除）的最新会话。
// 重用检测用：传入的 token 已删除时，查 family 是否还有活跃 session。
// 返回 nil 表示 family 下已无活跃会话（正常过期/已登出）。
func (ss *UserSessionStore) GetActiveByFamily(familyID string) (*model.UserSession, error) {
	if familyID == "" {
		return nil, nil
	}
	var us model.UserSession
	err := ss.s.Query(
		`SELECT * FROM user_sessions WHERE family_id = :family_id
		 ORDER BY id DESC LIMIT 1`,
		map[string]any{"family_id": familyID},
	).Get(&us)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &us, nil
}

// Rotate 轮换 refresh token：删旧 token + 建同 family 新 token，事务内执行。
//   - oldToken：被轮换掉的旧 token 哈希
//   - newToken：新 token 哈希
//   - newExpiresAt：新 token 的滑动过期时间（调用方已与绝对上限取 min）
//
// 返回新会话记录。调用方据此签发 access JWT（需 newSess.ID 作为 sid）。
func (ss *UserSessionStore) Rotate(oldToken, newToken string, newExpiresAt time.Time) (*model.UserSession, error) {
	var newSess *model.UserSession
	err := ss.s.Transaction(func(bs *base.Session) error {
		t := &Session{Session: bs} // 包装回 store.Session 以便用 Query
		// 查旧 session 拿 family/user/absolute_expires 等信息
		var old model.UserSession
		err := t.Query(
			`SELECT * FROM user_sessions WHERE token = :token`,
			map[string]any{"token": oldToken},
		).Get(&old)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("session to rotate not found")
		}
		if err != nil {
			return err
		}

		// 删旧 token
		if _, err := t.Query(
			`DELETE FROM user_sessions WHERE token = :token`,
			map[string]any{"token": oldToken},
		).Exec(); err != nil {
			return err
		}

		// 建同 family 新 token，复用原 user_id / absolute_expires_at / ua / ip
		_, err = t.Query(
			`INSERT INTO user_sessions (token, family_id, user_id, expires_at, absolute_expires_at, ua, ip)
			 VALUES (:token, :family_id, :user_id, :expires_at, :absolute_expires_at, :ua, :ip)`,
			map[string]any{
				"token":               newToken,
				"family_id":           old.FamilyID,
				"user_id":             old.UserID,
				"expires_at":          newExpiresAt,
				"absolute_expires_at": old.AbsoluteExpiresAt,
				"ua":                  old.UA,
				"ip":                  old.IP,
			},
		).Exec()
		if err != nil {
			return err
		}

		// 查回新 session
		err = t.Query(
			`SELECT * FROM user_sessions WHERE token = :token`,
			map[string]any{"token": newToken},
		).Get(&old)
		if err != nil {
			return err
		}
		newSess = &old
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newSess, nil
}

// Delete 删除单条会话（按 token 哈希）。
func (ss *UserSessionStore) Delete(token string) error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions WHERE token = :token`,
		map[string]any{"token": token},
	).Exec()
	return err
}

// DeleteByFamily 吊销整个登录链（同 family_id 的所有会话）。
// 登出当前设备 / 检测到重用攻击时调用。
func (ss *UserSessionStore) DeleteByFamily(familyID string) error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions WHERE family_id = :family_id`,
		map[string]any{"family_id": familyID},
	).Exec()
	return err
}

// DeleteByUser 吊销用户所有设备的会话。
// 改密 / 禁用用户 / 重置密码时调用，强制全部重登。
func (ss *UserSessionStore) DeleteByUser(userID int64) error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions WHERE user_id = :user_id`,
		map[string]any{"user_id": userID},
	).Exec()
	return err
}

// DeleteExpired 清理已过期会话（滑动过期或绝对过期）。
// 启动时调用一次。
func (ss *UserSessionStore) DeleteExpired() error {
	_, err := ss.s.Query(
		`DELETE FROM user_sessions
		 WHERE expires_at < datetime('now','localtime')
		    OR absolute_expires_at < datetime('now','localtime')`,
		nil,
	).Exec()
	return err
}
