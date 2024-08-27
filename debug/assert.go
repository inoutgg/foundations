//go:build !production && !noassert

package debug

import "fmt"

// Assert checks if a condition is true. If not, it will panic.
//
// Assert calls are ignore when "noassert" build tag is provided.
func Assert(condition bool, m string, args ...any) {
	if !condition {
		panic(fmt.Errorf(m, args...))
	}
}
