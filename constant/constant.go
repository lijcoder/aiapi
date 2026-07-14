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
)

func ParseArgs() {
	flag.StringVar(&ADDRESS, "address", ":", "http address")
	flag.IntVar(&PORT, "port", 8888, "http port")
	flag.BoolVar(&PPROF, "add-pprof", false, "add pprof")
	flag.IntVar(&MEMLIMIT, "mem", 20, "memory limit(MB)")
	flag.IntVar(&GCPERCENT, "gc", 100, "gc percent")
	flag.Parse()
}

func Address() string {
	return ADDRESS + strconv.Itoa(PORT)
}

// DBFilePath 返回数据库文件路径：~/.aiapi/aiapi.db
func DBFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aiapi", "aiapi.db")
}
