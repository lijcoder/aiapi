// Package router 注册反向代理的 HTTP 路由，是代理链路的 echo 框架适配层
// （与 manager/router 同构：路由在业务自己的包下配置，framework 只建路由组）。
package router

import (
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lijcoder/aiapi/proxy"
	"github.com/lijcoder/aiapi/proxy/types"
)

// echoResponseWrite 适配 echo 响应写入到 proxy 的 ProxyResponseWrite 接口
type echoResponseWrite struct {
	e echo.Context
}

func (w *echoResponseWrite) Header() http.Header {
	return w.e.Response().Header()
}

func (w *echoResponseWrite) WriteStatusCode(statusCode int) {
	w.e.Response().WriteHeader(statusCode)
}

func (w *echoResponseWrite) Write(body []byte) (int, error) {
	n, err := w.e.Response().Writer.Write(body)
	if err != nil {
		return n, err
	}
	if flusher, ok := w.e.Response().Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, nil
}

// Register 在已建好的 /proxy group 上注册代理路由。
//   - GET /:format/:provider/v1/models：模型列表元数据端点（具体路由，静态段优先于通配符 *），
//     非 GET 的同路径请求不匹配此路由，落回通配路由按转发请求处理
//   - ANY /:format/:provider/*：转发请求，透传到上游 Provider
func Register(g *echo.Group) {
	g.GET("/:format/:provider/v1/models", modelsProcess)
	g.Any("/:format/:provider/*", directProcess)
}

func directProcess(c echo.Context) error {
	req, err := buildProxyRequest(c)
	if err != nil {
		return err
	}
	return proxy.Handle(req)
}

func modelsProcess(c echo.Context) error {
	req, err := buildProxyRequest(c)
	if err != nil {
		return err
	}
	// 具体路由无通配参数，路径即 v1/models（供日志记录）
	req.Path = "v1/models"
	return proxy.HandleModels(req)
}

// buildProxyRequest 从 echo 上下文提取代理请求参数
func buildProxyRequest(c echo.Context) (types.ProxyRequest, error) {
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return types.ProxyRequest{}, err
	}
	return types.ProxyRequest{
		Format:    c.Param("format"),
		Provider:  c.Param("provider"),
		Method:    c.Request().Method,
		Path:      c.Param("*"),
		Body:      bodyBytes,
		Query:     c.QueryParams(),
		Writer:    &echoResponseWrite{e: c},
		Headers:   c.Request().Header,
		StartTime: time.Now(),
		// 框架适配：从 echo 提取标准库 context 注入 proxy 层。
		// 客户端断开时 echo/http.Server 会取消它，proxy 据此取消上游请求。
		// 对接其他框架时，在对应适配层提供同样的 context 即可。
		Ctx: c.Request().Context(),
	}, nil
}
