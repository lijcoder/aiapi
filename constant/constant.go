package constant

import (
	"flag"
	"strconv"
)

var (
	ADDRESS = ":"
	PORT    = 8888
)

func ParseAgrs() {
	argAddress := flag.String("address", ":", "http address")
	argPort := flag.Int("port", 8888, "http port")
	flag.Parse()

	ADDRESS = *argAddress
	PORT = *argPort
}

func Address() string {
	return ADDRESS + strconv.Itoa(PORT)
}
