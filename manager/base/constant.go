package base

import "time"

// 后台管理相关常量。manager 功能自包含在 manager 包内，路由组根路径在 framework/echo.go 注册。
// 注意：跨业务复用的基础设施常量（密钥文件、环境变量名等）放根目录 constant/ 包，
// 这里只放 manager 域自用的常量。
const (
	CookieName = "refresh_token" // 登录态 cookie 名（存 refresh token）

	CookiePath = "/manager" // cookie 作用路径，限定管理接口

	// Refresh token 有效期（滑动续期）
	SessionTTL         = 7 * 24 * time.Hour  // refresh 滑动窗口：每次刷新顺延至此
	SessionAbsoluteTTL = 30 * 24 * time.Hour // refresh 绝对上限：登录时设定，不随刷新顺延

	AccessTTL = 15 * time.Minute // access JWT 有效期

	TokenBytes = 32 // refresh token 随机字节数（hex 后 64 字符）

	JWTIssuer = "aiapi" // access JWT 签发者
)
