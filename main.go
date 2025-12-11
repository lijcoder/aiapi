package main

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/framework"
)

func main() {
	constant.ParseAgrs()
	slog.SetLogLoggerLevel(slog.LevelInfo)
	e := echo.New()
	framework.EchoInit(e)
	e.Logger.Fatal(e.Start(constant.Address()))
}
