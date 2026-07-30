// crypto.go 提供「敏感配置落库加密」的通用能力（AES-256-GCM）。
//
// 加密密钥由 CryptoSecret（见 manager/base/secret.go）按用途派生：
// SHA-256(CryptoSecret + purpose)，不同业务用不同 purpose 实现密钥隔离
// （互不能解密对方的密文）。输出格式：base64(nonce + ciphertext)，
// 密文可区分于明文 JSON（明文以 '{' 开头）。
//
// 注意：轮换加密密钥会导致所有密文无法解密（TOTP 密钥、provider 配置等），
// 轮换需配套 re-encrypt 迁移，运维文档需明确该影响。
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/lijcoder/aiapi/manager/base"
)

// 加密用途常量：所有 purpose 集中定义于此，禁止在各业务处散写字符串字面量。
// 不同用途派生不同密钥，实现业务间密钥隔离（互不能解密对方密文）。
// 新增加密场景时在此追加常量，命名对齐业务语义。
const (
	purposeTOTPSecret     = ":totp-secret"     // TOTP 密钥（users.totp_secret）
	purposeProviderConfig = ":provider-config" // Provider 配置（providers.config，含上游 Key）
	purposeAPIKey         = ":api-key"         // API Key 原文（api_keys.key_enc，可还原查看）
)

// deriveKey 按用途派生 AES-256 密钥：SHA-256(CryptoSecret + purpose)。
func deriveKey(purpose string) ([]byte, error) {
	if len(base.CryptoSecret) == 0 {
		return nil, errors.New("crypto secret not loaded")
	}
	h := sha256.Sum256(append([]byte(base.CryptoSecret), purpose...))
	return h[:], nil
}

// encryptWithPurpose 用指定用途的派生密钥加密明文，输出 base64(nonce + ciphertext)。
func encryptWithPurpose(plain, purpose string) (string, error) {
	key, err := deriveKey(purpose)
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

// decryptWithPurpose 用指定用途的派生密钥解密密文。
func decryptWithPurpose(enc, purpose string) (string, error) {
	key, err := deriveKey(purpose)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
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
		return "", errors.New("ciphertext too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt ciphertext: %w", err)
	}
	return string(plain), nil
}
