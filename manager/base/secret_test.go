package base

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lijcoder/aiapi/constant"
)

// withTempDataDir 将 DataDir 指向临时目录，测试后恢复
func withTempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := constant.DataDir
	constant.DataDir = dir
	t.Cleanup(func() { constant.DataDir = old })
	return dir
}

func clearSecretEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"AIAPI_JWT_SECRET", "AIAPI_CRYPTO_SECRET"} {
		old, ok := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if ok {
				os.Setenv(k, old)
			}
		})
	}
}

func TestLoadSecrets_EnvPriority(t *testing.T) {
	withTempDataDir(t)
	clearSecretEnv(t)
	t.Setenv("AIAPI_JWT_SECRET", "env-jwt-secret-at-least-32-bytes-long!")
	t.Setenv("AIAPI_CRYPTO_SECRET", "env-crypto-secret-at-least-32-bytes!")

	if err := LoadSecrets(); err != nil {
		t.Fatalf("LoadSecrets err: %v", err)
	}
	if string(JWTSecret) != "env-jwt-secret-at-least-32-bytes-long!" {
		t.Fatal("jwt secret should come from env")
	}
	if string(CryptoSecret) != "env-crypto-secret-at-least-32-bytes!" {
		t.Fatal("crypto secret should come from env")
	}
}

func TestLoadSecrets_GenerateAndReuse(t *testing.T) {
	dir := withTempDataDir(t)
	clearSecretEnv(t)

	// 首次：无 env 无文件 → 自动生成并写文件
	if err := LoadSecrets(); err != nil {
		t.Fatalf("first LoadSecrets err: %v", err)
	}
	jwt1, crypto1 := string(JWTSecret), string(CryptoSecret)
	if len(jwt1) < constant.MinSecretLen || len(crypto1) < constant.MinSecretLen {
		t.Fatal("generated secrets too short")
	}
	if jwt1 == crypto1 {
		t.Fatal("fresh install should generate two independent secrets")
	}

	// 文件应存在且权限 0600
	for _, f := range []string{"jwt.key", "crypto.key"} {
		fi, err := os.Stat(filepath.Join(dir, "keys", f))
		if err != nil {
			t.Fatalf("key file %s not created: %v", f, err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("key file %s perm should be 0600, got %o", f, fi.Mode().Perm())
		}
	}

	// 第二次加载：应从文件读出相同值（不重新生成）
	JWTSecret, CryptoSecret = nil, nil
	if err := LoadSecrets(); err != nil {
		t.Fatalf("second LoadSecrets err: %v", err)
	}
	if string(JWTSecret) != jwt1 || string(CryptoSecret) != crypto1 {
		t.Fatal("secrets should be reused from key files, not regenerated")
	}
}

func TestLoadSecrets_CryptoSeededFromJWT(t *testing.T) {
	withTempDataDir(t)
	clearSecretEnv(t)
	t.Setenv("AIAPI_JWT_SECRET", "legacy-jwt-secret-at-least-32-bytes!!")

	// 兼容场景：只有 JWT 密钥（旧部署升级），crypto.key 不存在
	// → 加密密钥应从签名密钥播种，保证旧密文（派生自 JWTSecret）仍可解密
	if err := LoadSecrets(); err != nil {
		t.Fatalf("LoadSecrets err: %v", err)
	}
	if string(CryptoSecret) != "legacy-jwt-secret-at-least-32-bytes!!" {
		t.Fatal("crypto secret should be seeded from jwt secret for legacy compatibility")
	}
}

func TestLoadSecrets_ShortEnvRejected(t *testing.T) {
	withTempDataDir(t)
	clearSecretEnv(t)
	t.Setenv("AIAPI_JWT_SECRET", "too-short")

	if err := LoadSecrets(); err == nil {
		t.Fatal("short env secret should be rejected")
	}
}
