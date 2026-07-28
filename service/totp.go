// totp.go 实现 TOTP 双因素认证（2FA）：
//   - 密钥生成 / 二维码 / 验证码校验（pquerna/otp）
//   - 密钥 AES-GCM 加密落库（密钥派生自 AIAPI_JWT_SECRET，DB 泄露不直接丢 2FA）
//   - 两步登录的 pending 票据、绑定的 setup 票据（HS256 短效 JWT，5 分钟）
//   - TOTP 码失败次数保护（内存计数，防 6 位码爆破）
//
// 对外仅暴露 TOTPService，加密/票据细节不导出。
package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"sync"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP 相关哨兵错误，handler 据此映射业务码
var (
	ErrTOTPInvalidTicket = errors.New("invalid or expired 2fa ticket") // 票据无效/过期/用途不符
	ErrTOTPCodeWrong     = errors.New("wrong totp code")               // 验证码错误
	ErrTOTPTooManyFails  = errors.New("too many totp failures")        // 失败次数过多，需重新登录
)

const (
	ticketPurposePending = "2fa_pending" // 登录第二步：密码已过，待 TOTP 验证
	ticketPurposeSetup   = "2fa_setup"   // 绑定流程：已生成密钥，待首个验证码确认

	ticketTTL      = 5 * time.Minute // 票据有效期
	totpMaxFails   = 5               // 同一票据允许的最大失败次数
	totpIssuerName = "aiapi"
)

// TOTPSetup 是绑定流程第一步返回给前端的数据。
type TOTPSetup struct {
	Secret      string // Base32 密钥（用户手动录入用）
	OtpauthURL  string // otpauth:// 链接（备用）
	QRCodeData  string // data:image/png;base64,... 二维码图片
	SetupTicket string // 绑定票据，确认时回传
	ExpiresIn   int    // 票据有效期（秒）
}

// TOTPService 封装 TOTP 业务：生成密钥、校验验证码、密钥加解密、票据签发。
// 无状态，进程内可共享单例。
type TOTPService struct {
	mu       sync.Mutex
	failures map[string]int // pending 票据哈希 → 连续失败次数（防 6 位码爆破）
}

// NewTOTPService 创建 TOTPService。
func NewTOTPService() *TOTPService {
	return &TOTPService{failures: map[string]int{}}
}

// GenerateSetup 为用户生成新的 TOTP 密钥与二维码，返回展示数据 + setup 票据。
// 密钥此时不落库：确认接口验证首个验证码通过后才加密入库（防误绑/截胡）。
func (s *TOTPService) GenerateSetup(userID int64, account string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuerName,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
	})
	if err != nil {
		return nil, err
	}

	// 二维码 PNG → data URI
	img, err := key.Image(220, 220)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	qrData := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	// setup 票据内含明文密钥（用户本人在绑定页本就能看到密钥），
	// 签名保证不可篡改，5 分钟过期。
	ticket, err := signTicket(userID, ticketPurposeSetup, key.Secret())
	if err != nil {
		return nil, err
	}
	return &TOTPSetup{
		Secret:      key.Secret(),
		OtpauthURL:  key.URL(),
		QRCodeData:  qrData,
		SetupTicket: ticket,
		ExpiresIn:   int(ticketTTL.Seconds()),
	}, nil
}

// ConfirmSetup 校验 setup 票据 + 首个验证码，通过则返回应落库的加密密钥。
// 由 handler 负责写库。
func (s *TOTPService) ConfirmSetup(userID int64, setupTicket, code string) (encSecret string, err error) {
	claims, err := parseTicket(setupTicket)
	if err != nil || claims.Sub != userID || claims.Purpose != ticketPurposeSetup || claims.Secret == "" {
		return "", ErrTOTPInvalidTicket
	}
	if !totp.Validate(code, claims.Secret) {
		return "", ErrTOTPCodeWrong
	}
	return encryptSecret(claims.Secret)
}

// SignPending 密码验证通过后签发登录 pending 票据（5 分钟有效，仅用于换 TOTP 验证）。
func (s *TOTPService) SignPending(userID int64) (string, error) {
	return signTicket(userID, ticketPurposePending, "")
}

// PeekPendingUser 解析 pending 票据返回用户 ID。
// 用于先查出该用户的加密密钥；票据用途/签名的全量校验在 VerifyLogin 内完成。
func (s *TOTPService) PeekPendingUser(pendingTicket string) (int64, error) {
	claims, err := parseTicket(pendingTicket)
	if err != nil || claims.Purpose != ticketPurposePending {
		return 0, ErrTOTPInvalidTicket
	}
	return claims.Sub, nil
}

// VerifyLogin 校验 pending 票据 + TOTP 验证码，全部通过返回用户 ID。
// 失败计数：同一票据连续失败 totpMaxFails 次后票据作废（需重新过密码登录）。
func (s *TOTPService) VerifyLogin(pendingTicket, code string, encSecret string) (int64, error) {
	claims, err := parseTicket(pendingTicket)
	if err != nil || claims.Purpose != ticketPurposePending {
		return 0, ErrTOTPInvalidTicket
	}

	ticketKey := hashToken(pendingTicket)
	if s.fails(ticketKey) >= totpMaxFails {
		return 0, ErrTOTPTooManyFails
	}

	secret, err := decryptSecret(encSecret)
	if err != nil {
		return 0, err
	}
	if !totp.Validate(code, secret) {
		s.incrFails(ticketKey)
		if s.fails(ticketKey) >= totpMaxFails {
			return 0, ErrTOTPTooManyFails
		}
		return 0, ErrTOTPCodeWrong
	}

	s.clearFails(ticketKey)
	return claims.Sub, nil
}

// ===== 失败计数（内存态，重启清零；票据本身只有 5 分钟寿命，无需持久化）=====

func (s *TOTPService) fails(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failures[key]
}

func (s *TOTPService) incrFails(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures[key]++
	// 顺手惰性清理：map 过大时清空（条目对应的都是 5 分钟短效票据，清空无害）
	if len(s.failures) > 10000 {
		s.failures = map[string]int{}
	}
}

func (s *TOTPService) clearFails(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, key)
}

// ===== 票据（HS256 短效 JWT，结构与 access JWT 相同，多 purpose/secret 声明）=====

type ticketClaims struct {
	Sub     int64  `json:"sub"`
	Exp     int64  `json:"exp"`
	Iat     int64  `json:"iat"`
	Iss     string `json:"iss"`
	Purpose string `json:"purpose"`
	Secret  string `json:"secret,omitempty"` // 仅 setup 票据携带
}

func signTicket(userID int64, purpose, secret string) (string, error) {
	if len(base.JWTSecret) == 0 {
		return "", errors.New("jwt secret not loaded")
	}
	exp := time.Now().Add(ticketTTL)
	payload, err := json.Marshal(ticketClaims{
		Sub:     userID,
		Exp:     exp.Unix(),
		Iat:     time.Now().Unix(),
		Iss:     base.JWTIssuer,
		Purpose: purpose,
		Secret:  secret,
	})
	if err != nil {
		return "", err
	}
	signingInput := jwtHeader + "." + b64(payload)
	mac := hmac.New(sha256.New, base.JWTSecret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64(mac.Sum(nil)), nil
}

func parseTicket(token string) (*ticketClaims, error) {
	if len(base.JWTSecret) == 0 {
		return nil, errors.New("jwt secret not loaded")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTOTPInvalidTicket
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, base.JWTSecret)
	mac.Write([]byte(signingInput))
	if !hmac.Equal([]byte(b64(mac.Sum(nil))), []byte(parts[2])) {
		return nil, ErrTOTPInvalidTicket
	}
	payloadBytes, err := b64Decode(parts[1])
	if err != nil {
		return nil, ErrTOTPInvalidTicket
	}
	var c ticketClaims
	if err := json.Unmarshal(payloadBytes, &c); err != nil {
		return nil, ErrTOTPInvalidTicket
	}
	if time.Now().Unix() > c.Exp || c.Iss != base.JWTIssuer {
		return nil, ErrTOTPInvalidTicket
	}
	return &c, nil
}

// ===== 密钥加解密（AES-256-GCM，密钥由 JWTSecret 派生）=====

// totpKey 派生加密密钥：SHA-256(JWTSecret + 用途后缀)，与 JWT 签名密钥隔离。
func totpKey() ([]byte, error) {
	if len(base.JWTSecret) == 0 {
		return nil, errors.New("jwt secret not loaded")
	}
	h := sha256.Sum256(append([]byte(base.JWTSecret), ":totp-secret"...))
	return h[:], nil
}

// encryptSecret 加密 TOTP 明文密钥，输出 base64(nonce + ciphertext)。
func encryptSecret(plain string) (string, error) {
	key, err := totpKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// decryptSecret 解密落库的 TOTP 密钥。
func decryptSecret(enc string) (string, error) {
	key, err := totpKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", fmt.Errorf("decode totp secret: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("totp secret too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt totp secret: %w", err)
	}
	return string(plain), nil
}
