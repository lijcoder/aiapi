package service

import (
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestEncryptDecryptSecret(t *testing.T) {
	setSecretForTest()
	const plain = "JBSWY3DPEHPK3PXP"

	enc, err := encryptSecret(plain)
	if err != nil {
		t.Fatalf("encryptSecret err: %v", err)
	}
	if enc == plain {
		t.Fatal("ciphertext should differ from plaintext")
	}
	dec, err := decryptSecret(enc)
	if err != nil {
		t.Fatalf("decryptSecret err: %v", err)
	}
	if dec != plain {
		t.Fatalf("roundtrip mismatch: got %q", dec)
	}

	// 篡改密文应解密失败（GCM 认证）
	tampered := enc[:len(enc)-4] + "AAAA"
	if _, err := decryptSecret(tampered); err == nil {
		t.Fatal("tampered ciphertext should fail decryption")
	}
}

func TestTicketSignParse(t *testing.T) {
	setSecretForTest()
	tok, err := signTicket(7, ticketPurposePending, "")
	if err != nil {
		t.Fatalf("signTicket err: %v", err)
	}
	c, err := parseTicket(tok)
	if err != nil {
		t.Fatalf("parseTicket err: %v", err)
	}
	if c.Sub != 7 || c.Purpose != ticketPurposePending {
		t.Fatalf("claims mismatch: %+v", c)
	}

	// 篡改签名应失败
	if _, err := parseTicket(tok[:len(tok)-2] + "xx"); err == nil {
		t.Fatal("tampered ticket should fail")
	}
}

func TestConfirmSetup(t *testing.T) {
	setSecretForTest()
	svc := NewTOTPService()

	setup, err := svc.GenerateSetup(9, "tester")
	if err != nil {
		t.Fatalf("GenerateSetup err: %v", err)
	}
	if setup.Secret == "" || setup.SetupTicket == "" || setup.QRCodeData == "" {
		t.Fatal("setup data incomplete")
	}

	// 用当前时间窗口生成合法验证码
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode err: %v", err)
	}
	enc, err := svc.ConfirmSetup(9, setup.SetupTicket, code)
	if err != nil {
		t.Fatalf("ConfirmSetup err: %v", err)
	}
	if enc == "" {
		t.Fatal("encrypted secret should not be empty")
	}

	// 错误验证码应拒绝
	if _, err := svc.ConfirmSetup(9, setup.SetupTicket, "000000"); !errors.Is(err, ErrTOTPCodeWrong) {
		t.Fatalf("wrong code should return ErrTOTPCodeWrong, got %v", err)
	}
	// 他人票据（userID 不符）应拒绝
	if _, err := svc.ConfirmSetup(10, setup.SetupTicket, code); !errors.Is(err, ErrTOTPInvalidTicket) {
		t.Fatalf("other user's ticket should return ErrTOTPInvalidTicket, got %v", err)
	}
}

func TestVerifyLoginFailCounting(t *testing.T) {
	setSecretForTest()
	svc := NewTOTPService()

	setup, _ := svc.GenerateSetup(3, "u3")
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	enc, err := svc.ConfirmSetup(3, setup.SetupTicket, code)
	if err != nil {
		t.Fatalf("ConfirmSetup err: %v", err)
	}

	pending, err := svc.SignPending(3)
	if err != nil {
		t.Fatalf("SignPending err: %v", err)
	}

	// 连续错误验证码：前 totpMaxFails-1 次返回 ErrTOTPCodeWrong，第 totpMaxFails 次后作废
	for i := 0; i < totpMaxFails-1; i++ {
		if _, err := svc.VerifyLogin(pending, "000000", enc); !errors.Is(err, ErrTOTPCodeWrong) {
			t.Fatalf("attempt %d should be ErrTOTPCodeWrong, got %v", i+1, err)
		}
	}
	if _, err := svc.VerifyLogin(pending, "000000", enc); !errors.Is(err, ErrTOTPTooManyFails) {
		t.Fatalf("attempt %d should be ErrTOTPTooManyFails, got %v", totpMaxFails, err)
	}
	// 即使随后给出正确码，票据也已作废
	if _, err := svc.VerifyLogin(pending, code, enc); !errors.Is(err, ErrTOTPTooManyFails) {
		t.Fatalf("after lockout even correct code should fail, got %v", err)
	}
}

func TestVerifyLoginSuccess(t *testing.T) {
	setSecretForTest()
	svc := NewTOTPService()

	setup, _ := svc.GenerateSetup(5, "u5")
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	enc, _ := svc.ConfirmSetup(5, setup.SetupTicket, code)

	pending, _ := svc.SignPending(5)
	uid, err := svc.VerifyLogin(pending, code, enc)
	if err != nil {
		t.Fatalf("VerifyLogin err: %v", err)
	}
	if uid != 5 {
		t.Fatalf("uid mismatch: %d", uid)
	}

	// setup 票据不能当 pending 票据用
	if _, err := svc.VerifyLogin(setup.SetupTicket, code, enc); !errors.Is(err, ErrTOTPInvalidTicket) {
		t.Fatalf("setup ticket used as pending should fail, got %v", err)
	}
}
