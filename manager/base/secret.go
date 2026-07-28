// secret.go 密钥管理：签名密钥（JWTSecret）与加密密钥（CryptoSecret）的加载编排。
//
// 职责拆分（为何两把钥匙）：
//   - JWTSecret   只用于签名（access JWT、2FA 票据），轮换代价低（用户重登即可）
//   - CryptoSecret 只用于派生落库加密密钥（TOTP 密钥、Provider 配置等），
//     轮换需 re-encrypt 迁移，几乎不换
//   混在一起会导致「签名密钥因加密数据不敢轮换」，拆分后各自独立。
//
// 单个密钥的加载逻辑（环境变量 > 密钥文件 > 生成并写文件）是通用基础设施，
// 统一由 constant.LoadSecret 实现，此处只做两把钥匙的编排与兼容播种。
// 密钥文件位于 <DataDir>/keys/（0600 权限），备份 DB 时必须排除该目录。
package base

import (
	"github.com/lijcoder/aiapi/constant"
)

// JWTSecret 是签名密钥：access JWT、2FA 票据的 HMAC-SHA256 签名。
var JWTSecret []byte

// CryptoSecret 是加密派生密钥：service/crypto.go 按用途派生 AES-256 密钥，
// 用于 TOTP 密钥、Provider 配置等敏感字段的落库加密。
var CryptoSecret []byte

// LoadSecrets 加载签名密钥与加密密钥，应在启动时（ParseArgs 之后）调用一次。
//
// 加密密钥的兼容逻辑：crypto.key 不存在时，若签名密钥来自环境变量
// （旧版本部署的唯一密钥来源，其历史密文派生自 JWTSecret），
// 则从签名密钥播种以保证历史密文仍可解密；全新部署则独立随机生成，两把钥匙天然隔离。
func LoadSecrets() error {
	jwt, jwtSrc, err := constant.LoadSecret(constant.EnvJWTSecret, constant.JWTKeyFilePath(), nil)
	if err != nil {
		return err
	}
	JWTSecret = jwt

	var seed []byte
	if jwtSrc == "env" {
		seed = jwt // 旧部署升级：历史密文派生自 env 中的 JWTSecret，播种保持可解
	}
	crypto, _, err := constant.LoadSecret(constant.EnvCryptoSecret, constant.CryptoKeyFilePath(), seed)
	if err != nil {
		return err
	}
	CryptoSecret = crypto
	return nil
}
