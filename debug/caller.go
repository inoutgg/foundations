package debug

import (
	"fmt"
	"path/filepath"
	"runtime"
)

//go:noinline
func caller(skip int) string {
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	cf := runtime.CallersFrames([]uintptr{pcs[0]})
	f, _ := cf.Next()

	var caller string
	if f.File != "" {
		file, line := f.File, f.Line
		dir, file := filepath.Split(file)

		return fmt.Sprintf("%s:%d", filepath.Join(filepath.Base(dir), file), line)
	}

	return caller
}
