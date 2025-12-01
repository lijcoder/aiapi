package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/messages/gemini"
)

func main() {
	e := echo.New()
	e.POST("/proxy/gemini/*", func(c echo.Context) error {
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
	e.Logger.Fatal(e.Start(":8080"))
}
