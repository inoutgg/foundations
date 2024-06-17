package main

import (
	"net/http"
	_ "net/http/pprof"

	"go.inout.gg/common/debug"
)

var d = debug.Debuglog("main")

func main() {
	d("hello, %s", "world")

	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}
