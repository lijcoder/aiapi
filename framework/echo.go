package framework

import (
	"database/sql"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lijcoder/aiapi/parser"
	"github.com/lijcoder/aiapi/proxy"
	"github.com/lijcoder/aiapi/proxy/types"
)

type EchoProxyDirectResponseWrite struct {
	E echo.Context
}

func (ep *EchoProxyDirectResponseWrite) Header() http.Header {
	return ep.E.Response().Header()
}

func (ep *EchoProxyDirectResponseWrite) WriteStatusCode(statusCode int) {
	ep.E.Response().WriteHeader(statusCode)
}

func (ep *EchoProxyDirectResponseWrite) Write(body []byte) (int, error) {
	len, err := ep.E.Response().Writer.Write(body)
	if err != nil {
		return len, err
	}
	// 尝试 Flush，不支持时忽略（如 Timeout 中间件包装的场景）
	if flusher, ok := ep.E.Response().Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return len, nil
}

func EchoInit(e *echo.Echo, db *sql.DB) {
	// 全局中间件
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("10M"))

	apiProxy(e, "/proxy", db)
}

func apiProxy(e *echo.Echo, group string, db *sql.DB) {
	proxyGroup := e.Group(group)
	proxyGroup.Any("/:format/:provider/*", func(c echo.Context) error {
		return proxyDirectProcess(c, db)
	})
}

// proxyDirectProcess 解析 HTTP 参数，交给 proxy 处理业务
func proxyDirectProcess(c echo.Context, database *sql.DB) error {
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	writer := &EchoProxyDirectResponseWrite{E: c}
	return proxy.Handle(types.ProxyRequest{
		DB:        database,
		Format:    c.Param("format"),
		Provider:  c.Param("provider"),
		Method:    c.Request().Method,
		Path:      c.Param("*"),
		Body:      bodyBytes,
		Query:     c.QueryParams(),
		Writer:    writer,
		ApiKey:    parser.ExtractApiKey(c.Request().Header, c.Param("format")),
		Headers:   c.Request().Header,
		StartTime: time.Now(),
	})
}
