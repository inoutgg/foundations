//go:build !production && !noassert

package debug

import "fmt"

// Assert checks if a condition is true. If not, it will panic.
//
// Assert calls are ignored unless "noassert" tag is provided when
// building the project.
func Assert(condition bool, m string, args ...any) {
	if !condition {
		panic(fmt.Errorf(m, args...))
	}
}
