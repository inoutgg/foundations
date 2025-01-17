//go:build !production && debug

package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

		var pcs [1]uintptr
		runtime.Callers(2, pcs[:]) // skip [Debuglog, closure]
		cf := runtime.CallersFrames([]uintptr{pcs[0]})
		f, _ := cf.Next()

		var caller string
		if f.File != "" {
			file, line := f.File, f.Line
			caller = format(file, line)
		}

		fmt.Println(fmt.Sprintf("%s %d [%s]: %s", tag, pid, caller, fmt.Sprintf(m, args...)))
	}
}

func format(file string, line int) string {
	dir, file := filepath.Split(file)
	return fmt.Sprintf("%s:%d", filepath.Join(filepath.Base(dir), file), line)
}
