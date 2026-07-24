package base

import (
	"os"
	"time"
)

// 后台管理相关常量。manager 功能自包含在 manager 包内，路由组根路径在 framework/echo.go 注册。
const (
	CookieName = "refresh_token" // 登录态 cookie 名（存 refresh token）

	CookiePath = "/manager" // cookie 作用路径，限定管理接口

	// Refresh token 有效期（滑动续期）
	SessionTTL        = 7 * 24 * time.Hour  // refresh 滑动窗口：每次刷新顺延至此
	SessionAbsoluteTTL = 30 * 24 * time.Hour // refresh 绝对上限：登录时设定，不随刷新顺延

	AccessTTL = 15 * time.Minute // access JWT 有效期

	TokenBytes = 32 // refresh token 随机字节数（hex 后 64 字符）

	JWTIssuer = "aiapi" // access JWT 签发者
)

// JWTSecret 是 access JWT 的 HMAC-SHA256 签名密钥。
// 公网部署必须通过环境变量 AIAPI_JWT_SECRET 提供（≥32 字节），启动时校验。
var JWTSecret []byte

// LoadJWTSecret 从环境变量读取 JWT 密钥，缺失或过短则返回错误。
// 应在应用启动时（ParseArgs 之后）调用一次。
func LoadJWTSecret() error {
	s := os.Getenv("AIAPI_JWT_SECRET")
	if len(s) < 32 {
		return ErrJWTSecretInvalid
	}
	JWTSecret = []byte(s)
	return nil
}
