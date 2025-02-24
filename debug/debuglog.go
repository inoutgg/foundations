//go:build !production && debug

package debug

import (
	"fmt"
	"os"
)

// Debuglog creates a print function that prints a message to stdout.
//
// It is a wrapper around fmt.Println.
//
// Debuglog calls are ignored unless debug tag is provided when
// building the project.
func Debuglog(tag string) func(string, ...any) {
	return func(m string, args ...any) {
		pid := os.Getpid()
		c := caller(3) // [caller, closure, Debuglog]

		fmt.Println(fmt.Sprintf("%s %d %s: %s", tag, pid, c, fmt.Sprintf(m, args...)))
	}
}
