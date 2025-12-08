package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/messages/gemini"
	"github.com/lijcoder/aiapi/proxy"
)

func main() {
	constant.ParseAgrs()
	e := echo.New()
	e.POST("/test/message/gemini/*", func(c echo.Context) error {
		req := new(gemini.Request)
		if err := c.Bind(req); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, req)
	})
	e.POST("/sse", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return c.String(http.StatusInternalServerError, "Streaming unsupported")
		}

		for i := range 10 {
			data := fmt.Sprintf("data: Message %d\n\n", i)
			if _, err := c.Response().Write([]byte(data)); err != nil {
				return err
			}
			flusher.Flush()
			time.Sleep(1 * time.Second)
		}

		return nil
	})
	e.POST("/proxy/:type/*", func(c echo.Context) error {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		req := proxy.ProxyDirectRequest{
			Type:        c.Param("type"),
			Path:        c.Param("*"),
			Method:      c.Request().Method,
			Headers:     c.Request().Header,
			QueryParams: c.QueryParams(),
			Body:        bodyBytes,
		}
		resp, err := proxy.Direct(req)
		if err != nil {
			return c.String(http.StatusBadGateway, err.Error())
		}
		defer resp.Body.Close()
		contentType := resp.Headers.Get("Content-Type")
		// Copy headers
		for k, vs := range resp.Headers {
			for _, v := range vs {
				c.Response().Header().Add(k, v)
			}
		}
		c.Response().Status = resp.StatusCode
		c.Response().WriteHeader(resp.StatusCode)
		sseContentType := "event-stream"
		// no stream
		if !strings.Contains(contentType, sseContentType) {
			bodyBytes, err = io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			_, err = c.Response().Writer.Write(bodyBytes)
			return err
		}
		// stream
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return c.String(http.StatusInternalServerError, "Streaming unsupported")
		}
		buf := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := c.Response().Writer.Write(buf[:n]); writeErr != nil {
					return writeErr
				}
				flusher.Flush()
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
		}
		return nil
	})
	e.Logger.Fatal(e.Start(constant.Address()))
}
