//go:build production || !debug

package debug

func Debuglog(_ string) func(string, ...any) { return func(_ string, _ ...any) { /*noop*/ } }
