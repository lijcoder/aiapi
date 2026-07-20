package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

// ServeFrontend 注册前端静态文件路由，提供 SPA fallback。
// 必须在所有 API 路由之后调用。
func ServeFrontend(e *echo.Echo) {
	dist, _ := fs.Sub(frontendDist, "frontend/dist")
	fsHandler := http.FileServerFS(dist)
	indexHTML := readIndex(dist)

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := dist.Open(r.URL.Path[1:])
		if err == nil {
			f.Close()
			fsHandler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})),
	)
}

func readIndex(dist fs.FS) []byte {
	f, err := dist.Open("index.html")
	if err != nil {
		return nil
	}
	defer f.Close()
	b := make([]byte, 65536)
	n, _ := f.Read(b)
	return b[:n]
}
