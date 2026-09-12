package main

import (
	"io"
	stdLog "log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/framework"
	aiapiLog "github.com/lijcoder/aiapi/log"
	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/driver"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	constant.ParseArgs()
	logOutput := initLogger()
	if err := base.LoadSecrets(); err != nil {
		slog.Error("secret load failed", "err", err)
		panic(err)
	}
	initStore()
	defer store.Close()

	slog.Info("server starting",
		"port", constant.PORT,
		"data-dir", constant.DataDir,
	)

	e := echo.New()
	e.HideBanner = true
	// Echo 在 New 时默认把 logger 写到 stdout；切换到与 slog 相同的
	// writer 后，Echo 和 http.Server 的内部日志也会落到 app.log。
	e.Logger.SetOutput(logOutput)
	e.StdLogger.SetOutput(logOutput)
	framework.EchoInit(e)
	ServeFrontend(e)

	// Echo 的 tcpKeepAliveListener 在 Accept 后设置 SO_KEEPALIVE；
	// macOS 上客户端刚断开时该 setsockopt 可能返回 EINVAL，
	// Echo 会把它当成致命监听错误。使用标准监听器可避免服务因此退出。
	listener, err := net.Listen("tcp", constant.Address())
	if err != nil {
		slog.Error("server listen failed", "address", constant.Address(), "err", err)
		os.Exit(1)
	}
	e.Listener = listener
	defer listener.Close()

	if err := e.StartServer(&http.Server{
		Addr:              constant.Address(),
		ReadTimeout:       time.Second * 10, // 客户端请求 body 需在 10s 内发完（仅约束读取阶段，不影响响应流）
		ReadHeaderTimeout: time.Second * 2,
		IdleTimeout:       time.Second * 90, // keep-alive 空闲连接存活时间（不设置则复用 ReadTimeout，空闲连接过早被关）
		// WriteTimeout 必须保持 0：它是「请求头读完 → 响应写完」的总时长死线，
		// 任何固定值都会切断超长 SSE 流式响应（LLM 流可能持续数分钟）。
		// 连接生命周期改由请求 context 控制（客户端断开 → 取消上游请求）。
		WriteTimeout: 0,
	}); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
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

func initLogger() io.Writer {
	os.MkdirAll(constant.LogDir(), 0755)

	fileWriter := &lumberjack.Logger{
		Filename:   constant.LogFilePath(),
		MaxSize:    100,  // MB
		MaxAge:     30,   // 天
		MaxBackups: 10,   // 最多保留 10 个旧文件
		LocalTime:  true, // 使用本地时间命名
	}

	w := io.MultiWriter(os.Stdout, fileWriter)
	slog.SetDefault(slog.New(aiapiLog.NewFormatter(w, slog.LevelInfo)))
	stdLog.SetOutput(w)
	return w
}
