package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/framework"
	aiapiLog "github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/driver"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	constant.ParseArgs()
	initLogger()
	initStore()
	defer store.Close()

	slog.Info("server starting",
		"port", constant.PORT,
		"data-dir", constant.DataDir,
	)

	e := echo.New()
	e.HideBanner = true
	framework.EchoInit(e)
	ServeFrontend(e)
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

func initStore() {
	db, err := driver.NewSQLite(constant.DBFilePath())
	if err != nil {
		slog.Error("db driver init failed", "err", err)
		panic(err)
	}

	// 连接池配置
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := store.Init(db); err != nil {
		slog.Error("db store init failed", "err", err)
		panic(err)
	}
}

func initLogger() {
	logDir := constant.LogDir()
	os.MkdirAll(logDir, 0755)

	fileWriter := &lumberjack.Logger{
		Filename:   logDir + "/app.log",
		MaxSize:    100,  // MB
		MaxAge:     30,   // 天
		MaxBackups: 10,   // 最多保留 10 个旧文件
		LocalTime:  true, // 使用本地时间命名
	}

	w := io.MultiWriter(os.Stdout, fileWriter)
	slog.SetDefault(slog.New(aiapiLog.NewFormatter(w, slog.LevelInfo)))
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
