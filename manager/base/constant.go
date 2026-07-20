package base

import "time"

// 后台管理相关常量。manager 功能自包含在 manager 包内，路由组根路径在 framework/echo.go 注册。
const (
	CookieName = "token"           // 登录态 cookie 名
	CookiePath = "/"               // cookie 作用路径
	SessionTTL = 24 * time.Hour    // 登录会话有效期
	TokenBytes = 32                // 随机 token 字节数（hex 后 64 字符）
)
