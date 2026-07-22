package constant

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
)

var (
	ADDRESS   = ":"
	PORT      = 8888
	PPROF     = false
	MEMLIMIT  = 20
	GCPERCENT = 100
	DataDir   string // 数据根目录
)

func ParseArgs() {
	home, _ := os.UserHomeDir()
	flag.StringVar(&ADDRESS, "address", ":", "http address")
	flag.IntVar(&PORT, "port", 8888, "http port")
	flag.BoolVar(&PPROF, "add-pprof", false, "add pprof")
	flag.IntVar(&MEMLIMIT, "mem", 20, "memory limit(MB)")
	flag.IntVar(&GCPERCENT, "gc", 100, "gc percent")
	flag.StringVar(&DataDir, "data-dir", filepath.Join(home, ".aiapi"), "data root directory")
	flag.Parse()
}

func Address() string {
	return ADDRESS + strconv.Itoa(PORT)
}

// DBFilePath 返回数据库文件路径
func DBFilePath() string {
	return filepath.Join(DataDir, "db", "aiapi.db")
}

// LogDir 返回日志目录
func LogDir() string {
	return filepath.Join(DataDir, "logs")
}

// LogFilePath 返回日志文件完整路径
func LogFilePath() string {
	return filepath.Join(LogDir(), "app.log")
}
