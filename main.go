package main

import (
	"log/slog"
	"net/http"
	"net/http/pprof"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/db"
	"github.com/lijcoder/aiapi/framework"
)

func main() {
	constant.ParseArgs()
	slog.SetLogLoggerLevel(slog.LevelInfo)

	database, err := db.Init()
	if err != nil {
		slog.Error("db init failed", "err", err)
		panic(err)
	}
	defer database.Close()

	e := echo.New()
	framework.EchoInit(e, database)
	if constant.PPROF {
		registerRoutes(e)
	}
	registerRuntime()
	e.Logger.Fatal(e.StartServer(&http.Server{
		Addr:              constant.Address(),
		ReadTimeout:       time.Second * 5,
		ReadHeaderTimeout: time.Second * 2,
		WriteTimeout:      time.Second * 90,
	}))
}

func registerRuntime() {
	debug.SetMemoryLimit(int64(constant.MEMLIMIT * 1024 * 1024))
	debug.SetGCPercent(constant.GCPERCENT)
}

func registerRoutes(engine *echo.Echo) {
	router := engine.Group("")
	// 下面的路由根据要采集的数据需求注册，不用全都注册
	router.GET("/debug/pprof", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/allocs", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/block", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/goroutine", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/heap", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/mutex", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	router.GET("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	router.GET("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	router.GET("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	router.GET("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
}
