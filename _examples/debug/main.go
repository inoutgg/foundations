package main

import (
	_ "net/http/pprof"

	"go.inout.gg/foundations/debug"
)

var d = debug.Debuglog("main")

func main() {
	helloworld()
}
