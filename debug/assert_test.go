//go:build !production && !noassert

package debug

import "testing"

func TestAssert(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		Assert(true, "it works")
		assertPanic(t, func() {
			Assert(false, "it should panic")
		})
	})
}

func assertPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	f()
}
