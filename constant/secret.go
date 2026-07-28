package constant

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// LoadSecret 按优先级加载单个密钥：环境变量 > 密钥文件 > 生成（或从 seed 播种）并写入文件。
//
// 这是密钥加载的通用基础设施，任何业务需要密钥时调用本方法，
// 不各自实现一套（避免文件权限、生成逻辑、播种兼容等行为不一致）。
//
// 参数：
//   - envName：环境变量名（如 EnvJWTSecret），设置时优先使用，长度需 ≥ MinSecretLen
//   - path：密钥文件完整路径（由 JWTKeyFilePath() 等方法提供），不存在时生成并写入（0600）
//   - seed：生成来源，非空（≥MinSecretLen）时直接播种为密钥（用于兼容场景），否则真随机生成
//
// 返回密钥内容与来源（env / file / generated）。
func LoadSecret(envName, path string, seed []byte) ([]byte, string, error) {
	// 1. 环境变量
	if s := os.Getenv(envName); s != "" {
		if len(s) < MinSecretLen {
			return nil, "", fmt.Errorf("%s 长度需至少 %d 字节", envName, MinSecretLen)
		}
		slog.Info("secret loaded", "env", envName, "source", "env")
		return []byte(s), "env", nil
	}

	// 2. 密钥文件
	if b, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(b))
		if len(s) < MinSecretLen {
			return nil, "", fmt.Errorf("密钥文件 %s 内容长度需至少 %d 字节", path, MinSecretLen)
		}
		slog.Info("secret loaded", "file", path, "source", "file")
		return []byte(s), "file", nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, "", fmt.Errorf("读取密钥文件 %s 失败: %w", path, err)
	}

	// 3. 生成：优先从 seed 播种（兼容场景），否则真随机生成
	var secret []byte
	if len(seed) >= MinSecretLen {
		secret = seed
		slog.Info("crypto secret seeded from jwt secret for legacy compatibility", "file", path)
	} else {
		b := make([]byte, MinSecretLen)
		if _, err := rand.Read(b); err != nil {
			return nil, "", fmt.Errorf("生成密钥失败: %w", err)
		}
		secret = []byte(hex.EncodeToString(b))
		slog.Info("secret generated", "file", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, "", fmt.Errorf("创建密钥目录失败: %w", err)
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, "", fmt.Errorf("写入密钥文件失败: %w", err)
	}
	// WriteFile 权限受 umask 影响，显式修正
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, "", fmt.Errorf("设置密钥文件权限失败: %w", err)
	}
	return secret, "generated", nil
}
