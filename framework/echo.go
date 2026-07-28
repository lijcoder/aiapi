package framework

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	managerRouter "github.com/lijcoder/aiapi/manager/router"
	proxyRouter "github.com/lijcoder/aiapi/proxy/router"
)

func EchoInit(e *echo.Echo) {
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("10M"))

	// 反向代理 API
	proxyGroup := e.Group("/proxy")
	proxyRouter.Register(proxyGroup)

	// 后台管理 API
	managerGroup := e.Group("/manager")
	managerRouter.Register(managerGroup)
}
