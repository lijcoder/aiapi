package service

import (
	"strings"
	"testing"
)

func TestEncryptWithPurpose_Roundtrip(t *testing.T) {
	setSecretForTest()
	const plain = `{"domain":"https://api.openai.com","headers":{"Authorization":["Bearer sk-real-key"]}}`

	enc, err := encryptWithPurpose(plain, purposeProviderConfig)
	if err != nil {
		t.Fatalf("encrypt err: %v", err)
	}
	if strings.Contains(enc, "sk-real-key") {
		t.Fatal("ciphertext should not contain plaintext")
	}
	dec, err := decryptWithPurpose(enc, purposeProviderConfig)
	if err != nil {
		t.Fatalf("decrypt err: %v", err)
	}
	if dec != plain {
		t.Fatalf("roundtrip mismatch: got %q", dec)
	}
}

func TestEncryptWithPurpose_PurposeIsolation(t *testing.T) {
	setSecretForTest()
	enc, _ := encryptWithPurpose("secret-data", purposeProviderConfig)

	// 用 TOTP 用途的密钥解 provider 密文，必须失败（密钥隔离）
	if _, err := decryptWithPurpose(enc, purposeTOTPSecret); err == nil {
		t.Fatal("decrypt with wrong purpose should fail")
	}
}

func TestDecryptProviderConfig_LegacyPlaintext(t *testing.T) {
	setSecretForTest()
	const legacy = `{"domain":"https://api.openai.com","headers":{}}`

	// 存量明文原样放行（渐进迁移）
	got, err := DecryptProviderConfig(legacy)
	if err != nil {
		t.Fatalf("legacy plaintext should pass through, got err: %v", err)
	}
	if got != legacy {
		t.Fatalf("legacy passthrough mismatch: %q", got)
	}

	// 新密文正常解密
	enc, _ := EncryptProviderConfig(legacy)
	got, err = DecryptProviderConfig(enc)
	if err != nil {
		t.Fatalf("decrypt err: %v", err)
	}
	if got != legacy {
		t.Fatalf("roundtrip mismatch: %q", got)
	}
}
