package framework

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/messages/gemini"
	"github.com/lijcoder/aiapi/proxy"
)

type EchoProxyDirectResponseWrite struct {
	E                echo.Context
	respWriteFlusher http.Flusher
}

func (ep *EchoProxyDirectResponseWrite) Header() http.Header {
	return ep.E.Response().Header()
}

func (ep *EchoProxyDirectResponseWrite) WriteStatusCode(statusCode int) {
	ep.E.Response().WriteHeader(statusCode)
}

func (ep *EchoProxyDirectResponseWrite) Write(body []byte) (int, error) {
	if ep.respWriteFlusher == nil {
		flusher, ok := ep.E.Response().Writer.(http.Flusher)
		if !ok {
			return 0, errors.New("http write flusher create fail")
		}
		ep.respWriteFlusher = flusher
	}

	len, err := ep.E.Response().Writer.Write((body))
	ep.respWriteFlusher.Flush()
	return len, err
}

// echo http set

type handlerFunc[T any] func(c echo.Context) (T, *constant.HttpCustomError)

func EchoInit(e *echo.Echo) {
	apiTest(e, "/test")
	apiManager(e, "/manager")
	apiProxy(e, "/proxy")
	apiProxyDebug(e, "/proxy/debug/:traceid")
}

func apiTest(e *echo.Echo, group string) {
	testGroup := e.Group(group)
	testGroup.POST("/message/gemini/*", debugMessage)
}

func apiManager(e *echo.Echo, group string) {
	// managerGroup := e.Group("/manager")
}

func apiProxy(e *echo.Echo, group string) {
	proxyGroup := e.Group(group)
	proxyGroup.Any("/route/*", proxyDirect)
	proxyGroup.Any("/direct/:type/*", proxyDirect)
}

func apiProxyDebug(e *echo.Echo, group string) {
	proxyGroup := e.Group(group)
	proxyGroup.Any("/direct/:type/*", proxyDirectDebug)
}

func GeneralHandler[T any](handlerFunc handlerFunc[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := handlerFunc(c)
		var resp constant.HttpGeneralResp
		if err != nil {
			slog.Error("http handler process error.", "api", c.Path(), "errStack", err)
			resp = constant.BuildHttpResponseFail(err.Msg)
		} else {
			resp = constant.BuildHttpResponseSuccess(result)
		}
		respErr := c.JSON(http.StatusOK, resp)
		if respErr != nil {
			slog.Error("http handler response error.", "api", c.Path(), "errStack", respErr)
		}
		return respErr
	}
}

// debug set
func debugMessage(c echo.Context) error {
	req := new(gemini.Request)
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, req)
}

// manager set

// proxy set
func proxyDirectDebug(c echo.Context) error {
	return proxyDirectProcess(c, true)
}

func proxyDirect(c echo.Context) error {
	return proxyDirectProcess(c, false)
}

func proxyDirectProcess(c echo.Context, debug bool) error {
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	pdr := &proxy.ProxyDirectRequest{
		Debug:       debug,
		TraceId:     c.Param("traceid"),
		Url:         c.Request().URL,
		Type:        c.Param("type"),
		Path:        c.Param("*"),
		Method:      c.Request().Method,
		Headers:     c.Request().Header,
		QueryParams: c.QueryParams(),
		Body:        bodyBytes,
	}
	pdw := EchoProxyDirectResponseWrite{E: c}
	p := proxy.ProxyDirect{Request: pdr, Response: &pdw}
	return p.Direct()
}
