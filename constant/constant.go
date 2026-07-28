package constant

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
)

var (
	ADDRESS   = ":"
	PORT      = 8888
	MEMLIMIT  = 20
	GCPERCENT = 100
	DataDir   string // 数据根目录
)

func ParseArgs() {
	home, _ := os.UserHomeDir()
	flag.StringVar(&ADDRESS, "address", ":", "http address")
	flag.IntVar(&PORT, "port", 8888, "http port")
	flag.IntVar(&MEMLIMIT, "mem", 20, "memory limit(MB)")
	flag.IntVar(&GCPERCENT, "gc", 100, "gc percent")
	flag.StringVar(&DataDir, "data-dir", filepath.Join(home, ".aiapi"), "data root directory")
	flag.Parse()
}

func Address() string {
	return ADDRESS + strconv.Itoa(PORT)
}

// DBFilePath 返回数据库文件路径
func DBFilePath() string {
	return filepath.Join(DataDir, "db", "aiapi.db")
}

// LogDir 返回日志目录
func LogDir() string {
	return filepath.Join(DataDir, "logs")
}

// LogFilePath 返回日志文件完整路径
func LogFilePath() string {
	return filepath.Join(LogDir(), "app.log")
}

// ===== 密钥基础设施（跨业务共享）=====
//
// 环境变量名、密钥文件名、长度下限是部署契约：未来新业务配置密钥时
// 必须复用这套定义，不各自造一套（避免配置重复/不当）。变更需同步文档。

const (
	MinSecretLen = 32 // 密钥长度下限（字节）

	EnvJWTSecret    = "AIAPI_JWT_SECRET"    // 签名密钥环境变量（JWT / 2FA 票据）
	EnvCryptoSecret = "AIAPI_CRYPTO_SECRET" // 加密密钥环境变量（落库敏感字段加密派生）

	jwtKeyFile    = "jwt.key"    // 签名密钥文件名
	cryptoKeyFile = "crypto.key" // 加密密钥文件名
)

// KeysDir 返回密钥文件目录（<DataDir>/keys，权限 0700）
func KeysDir() string {
	return filepath.Join(DataDir, "keys")
}

// JWTKeyFilePath 返回签名密钥文件完整路径
func JWTKeyFilePath() string {
	return filepath.Join(KeysDir(), jwtKeyFile)
}

// CryptoKeyFilePath 返回加密密钥文件完整路径
func CryptoKeyFilePath() string {
	return filepath.Join(KeysDir(), cryptoKeyFile)
}
