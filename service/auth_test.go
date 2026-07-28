package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
)

func setSecretForTest() {
	base.JWTSecret = []byte("test-secret-must-be-at-least-32-bytes-long!!")
	base.CryptoSecret = []byte("test-crypto-secret-at-least-32-bytes!")
}

func TestSignAndParseAccess(t *testing.T) {
	setSecretForTest()
	tok, exp, err := signAccess(42)
	if err != nil {
		t.Fatalf("signAccess err: %v", err)
	}
	if !exp.After(time.Now()) {
		t.Fatal("exp should be future")
	}
	// 结构: header.payload.sig
	if strings.Count(tok, ".") != 2 {
		t.Fatalf("token should have 2 dots, got: %s", tok)
	}
	c, err := parseAccess(tok)
	if err != nil {
		t.Fatalf("parseAccess err: %v", err)
	}
	if c.Sub != 42 || c.Iss != base.JWTIssuer {
		t.Fatalf("claims mismatch: %+v", c)
	}
}

func TestParseAccessTampered(t *testing.T) {
	setSecretForTest()
	tok, _, _ := signAccess(1)
	// 篡改 payload
	parts := strings.Split(tok, ".")
	parts[1] = parts[1] + "x"
	tampered := strings.Join(parts, ".")
	if _, err := parseAccess(tampered); err == nil {
		t.Fatal("tampered token should fail")
	}
	// 篡改签名
	parts = strings.Split(tok, ".")
	parts[2] = parts[2] + "x"
	tampered = strings.Join(parts, ".")
	if _, err := parseAccess(tampered); err == nil {
		t.Fatal("tampered signature should fail")
	}
}

func TestParseAccessWrongSecret(t *testing.T) {
	setSecretForTest()
	tok, _, _ := signAccess(1)
	// 换密钥
	base.JWTSecret = []byte("another-secret-must-be-at-least-32-bytes!!")
	if _, err := parseAccess(tok); err == nil {
		t.Fatal("different secret should fail")
	}
}

func TestParseAccessExpired(t *testing.T) {
	setSecretForTest()
	// 测试内部构造一个已过期的 JWT，不依赖产品码暴露的辅助函数
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"sub": 1, "exp": time.Now().Unix() - 1, "iat": time.Now().Unix() - 2, "iss": base.JWTIssuer,
	})
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, base.JWTSecret)
	mac.Write([]byte(signingInput))
	tok := signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	_, err := parseAccess(tok)
	if err != ErrTokenExpired {
		t.Fatalf("expired token should return ErrTokenExpired, got: %v", err)
	}
}

func TestBuildAndParseFamilyID(t *testing.T) {
	family := "abc123"
	random := "deadbeef"
	tok := buildRefreshToken(family, random)
	if got := parseFamilyID(tok); got != family {
		t.Fatalf("family mismatch: got %s want %s", got, family)
	}
	if !strings.Contains(tok, ".") {
		t.Fatal("refresh token should contain dot")
	}
}

func TestParseFamilyIDInvalid(t *testing.T) {
	if parseFamilyID("nodot") != "" {
		t.Fatal("invalid token should return empty family")
	}
	if parseFamilyID(".leading") != "" {
		t.Fatal("leading dot should return empty family")
	}
}

func TestHashToken(t *testing.T) {
	h1 := hashToken("abc")
	h2 := hashToken("abc")
	h3 := hashToken("abd")
	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	if h1 == h3 {
		t.Fatal("different input should produce different hash")
	}
}
