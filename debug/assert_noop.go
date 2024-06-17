//go:build production || noassert

package debug

func Assert(condition bool, m string, args ...any) { /*noop*/ }
