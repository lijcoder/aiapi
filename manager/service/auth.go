// Package service 封装管理台通用业务逻辑，不依赖 echo，供 handler/middleware 调用。
//
// auth.go 实现双 token 登录机制：
//   - access JWT（HS256，自实现，无状态，15min）
//   - refresh token（随机串，DB 持久化，滑动 7 天 / 绝对 30 天，轮换 + 重用检测）
//
// 对外仅暴露 SessionService 及其方法、TokenPair、若干哨兵错误。
// JWT 编解码、token 生成/哈希等均为内部实现，不导出。
package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ===== 对外哨兵错误 =====
//
// handler / middleware 据此映射 HTTP 业务码：
//   - ErrTokenExpired     → CodeTokenExpired（前端触发 /refresh）
//   - ErrSessionExpired   → CodeSessionExpired（清 cookie 跳登录）
//   - ErrSessionNotFound  → CodeSessionExpired（同上）
//   - ErrSessionReuse     → CodeSessionReuse（重用攻击，已吊销 family）
//   - ErrUserDisabled     → 用户被禁用/删除

var (
	ErrTokenExpired    = errors.New("token expired")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionReuse    = errors.New("session reuse detected")
	ErrUserDisabled    = errors.New("user disabled")
)

// ===== 对外类型 =====

// TokenPair 是签发/刷新后返回给 handler 的 token 组合。
type TokenPair struct {
	AccessToken  string    // access JWT，放响应体，前端存内存
	AccessExp    time.Time
	RefreshToken string    // 明文 refresh token，仅写 HttpOnly cookie
	RefreshExp   time.Time // cookie 过期时间
}

// SessionService 封装登录会话业务：签发、刷新（轮换+重用检测）、校验、吊销。
// 无状态，进程内可共享单例。
type SessionService struct{}

// NewSessionService 创建 SessionService。
func NewSessionService() *SessionService { return &SessionService{} }

// Issue 登录成功后签发 access + refresh。
//   - userID：登录用户 ID
//   - ua/ip：User-Agent 摘要与登录 IP，仅记录用于展示
//
// 返回 TokenPair：AccessToken 放响应体，RefreshToken 写 HttpOnly cookie。
func (s *SessionService) Issue(userID int64, ua, ip string) (*TokenPair, error) {
	familyID, err := newFamilyID()
	if err != nil {
		return nil, err
	}
	random, err := newToken()
	if err != nil {
		return nil, err
	}
	refreshToken := buildRefreshToken(familyID, random)

	now := time.Now()
	refreshExp := now.Add(base.SessionTTL)
	absoluteExp := now.Add(base.SessionAbsoluteTTL)

	// refresh token 哈希后存库
	if err := store.C().UserSession().Create(
		hashToken(refreshToken), familyID, userID, refreshExp, absoluteExp, ua, ip,
	); err != nil {
		return nil, err
	}

	access, accessExp, err := signAccess(userID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		AccessExp:    accessExp,
		RefreshToken: refreshToken,
		RefreshExp:   refreshExp,
	}, nil
}

// Refresh 刷新 access + 轮换 refresh，含重用检测。
//   - presentedRefresh：客户端传入的明文 refresh token（来自 cookie）
//
// 失败返回哨兵错误，handler 据此映射业务码。重用检测命中时已吊销整个 family。
func (s *SessionService) Refresh(presentedRefresh string) (*TokenPair, error) {
	if presentedRefresh == "" {
		return nil, ErrSessionNotFound
	}
	presentedHash := hashToken(presentedRefresh)

	// 1. 查传入的 token
	sess, err := store.C().UserSession().GetByToken(presentedHash)
	if err != nil {
		return nil, err
	}

	// 2. token 不存在 → 重用检测
	if sess == nil {
		familyID := parseFamilyID(presentedRefresh)
		if familyID != "" {
			active, err := store.C().UserSession().GetActiveByFamily(familyID)
			if err != nil {
				return nil, err
			}
			if active != nil {
				// 重用攻击：family 下还有活跃 session，却用了已删除的 token
				slog.Warn("refresh token reuse detected", "family_id", familyID)
				_ = store.C().UserSession().DeleteByFamily(familyID)
				return nil, ErrSessionReuse
			}
		}
		return nil, ErrSessionNotFound
	}

	// 3. token 存在但过期 → 删并返回
	now := time.Now()
	if sess.ExpiresAt.Before(now) || sess.AbsoluteExpiresAt.Before(now) {
		_ = store.C().UserSession().Delete(presentedHash)
		return nil, ErrSessionExpired
	}

	// 4. 校验用户仍可用
	user, err := store.C().User().GetByID(sess.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Enabled {
		_ = store.C().UserSession().DeleteByFamily(sess.FamilyID)
		return nil, ErrUserDisabled
	}

	// 5. 轮换：删旧 + 建新（同 family），事务内
	newRandom, err := newToken()
	if err != nil {
		return nil, err
	}
	newRefresh := buildRefreshToken(sess.FamilyID, newRandom)
	// 滑动续期：顺延至 now+SessionTTL，但不超过绝对上限
	newExp := now.Add(base.SessionTTL)
	if newExp.After(sess.AbsoluteExpiresAt) {
		newExp = sess.AbsoluteExpiresAt
	}
	if _, err = store.C().UserSession().Rotate(presentedHash, hashToken(newRefresh), newExp); err != nil {
		return nil, err
	}

	// 6. 签发新 access JWT
	access, accessExp, err := signAccess(user.ID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		AccessExp:    accessExp,
		RefreshToken: newRefresh,
		RefreshExp:   newExp,
	}, nil
}

// ValidateAccess 校验 access JWT，返回关联用户。
// 供 Auth 中间件使用：JWT 过期返回 ErrTokenExpired，用户被禁用返回 ErrUserDisabled。
func (s *SessionService) ValidateAccess(tokenStr string) (*model.User, error) {
	claims, err := parseAccess(tokenStr)
	if err != nil {
		return nil, err
	}
	user, err := store.C().User().GetByID(claims.Sub)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Enabled {
		return nil, ErrUserDisabled
	}
	return user, nil
}

// RevokeByRefreshToken 吊销 refresh token 对应的登录链。
// 用于登出：从 cookie 读 refresh → 吊销整个 family。
func (s *SessionService) RevokeByRefreshToken(refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	sess, err := store.C().UserSession().GetByToken(hashToken(refreshToken))
	if err != nil {
		return err
	}
	if sess == nil {
		return nil // 已过期/不存在，视为已吊销
	}
	return store.C().UserSession().DeleteByFamily(sess.FamilyID)
}

// RevokeByUser 吊销用户所有设备的登录会话。
// 用于改密 / 禁用用户 / 重置密码，强制全部重登。
func (s *SessionService) RevokeByUser(userID int64) error {
	return store.C().UserSession().DeleteByUser(userID)
}

// ===================================================================
// 以下为内部实现，不对外导出。JWT 编解码、token 生成/哈希等细节封装于此。
// ===================================================================

// ----- access JWT（HS256，仅用标准库）-----

// accessClaims 是 access JWT 的 payload。
type accessClaims struct {
	Sub int64  `json:"sub"` // user_id
	Exp int64  `json:"exp"` // 过期 unix 秒
	Iat int64  `json:"iat"` // 签发 unix 秒
	Iss string `json:"iss"` // 签发者
}

var jwtHeader = b64([]byte(`{"alg":"HS256","typ":"JWT"}`))

// signAccess 签发 access JWT，返回 token 字符串与过期时间。
func signAccess(userID int64) (string, time.Time, error) {
	if len(base.JWTSecret) == 0 {
		return "", time.Time{}, errors.New("jwt secret not loaded")
	}
	exp := time.Now().Add(base.AccessTTL)
	payload, err := json.Marshal(accessClaims{
		Sub: userID,
		Exp: exp.Unix(),
		Iat: time.Now().Unix(),
		Iss: base.JWTIssuer,
	})
	if err != nil {
		return "", time.Time{}, err
	}
	signingInput := jwtHeader + "." + b64(payload)
	mac := hmac.New(sha256.New, base.JWTSecret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64(mac.Sum(nil)), exp, nil
}

// parseAccess 校验并解析 access JWT。
// 校验失败（签名错/过期/签发者不符）返回非 nil error。
func parseAccess(token string) (*accessClaims, error) {
	if len(base.JWTSecret) == 0 {
		return nil, errors.New("jwt secret not loaded")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, base.JWTSecret)
	mac.Write([]byte(signingInput))
	// 恒定时间比较，防时序攻击
	if !hmac.Equal([]byte(b64(mac.Sum(nil))), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}
	payloadBytes, err := b64Decode(parts[1])
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}
	var c accessClaims
	if err := json.Unmarshal(payloadBytes, &c); err != nil {
		return nil, errors.New("invalid payload")
	}
	if time.Now().Unix() > c.Exp {
		return nil, ErrTokenExpired
	}
	if c.Iss != base.JWTIssuer {
		return nil, errors.New("invalid issuer")
	}
	return &c, nil
}

// b64 / b64Decode：base64url 编解码（无 padding，JWT 规范要求）
func b64(b []byte) string                { return base64.RawURLEncoding.EncodeToString(b) }
func b64Decode(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

// ----- refresh token 生成 / 哈希 / 解析 -----

// newToken 生成随机 hex token（32 字节 → 64 字符）。
func newToken() (string, error) {
	b := make([]byte, base.TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// newFamilyID 生成登录 family 标识（16 字节 → 32 字符 hex）。
// 每次登录生成新 family，重用检测按 family 吊销整个登录链。
func newFamilyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken 对 refresh token 做 SHA-256 哈希（hex 输出）。
// 数据库只存哈希，即使 DB 泄露攻击者也无法直接用 refresh token 登录。
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// buildRefreshToken 组装 refresh token：familyID.randomHex。
// familyID 前缀用于重用检测（token 被轮换删除后仍能解析出 family）。
func buildRefreshToken(familyID, randomHex string) string {
	return familyID + "." + randomHex
}

// parseFamilyID 从 refresh token 解析 familyID 前缀。格式 familyID.randomHex，失败返回空串。
func parseFamilyID(token string) string {
	idx := strings.IndexByte(token, '.')
	if idx <= 0 {
		return ""
	}
	return token[:idx]
}
