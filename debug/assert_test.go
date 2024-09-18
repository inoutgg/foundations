//go:build !production && !noassert

package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssert(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		Assert(true, "it works")
		assert.Panics(t, func() {
			Assert(false, "it should panic")
		}, "Expected Assert(false, ...) to panic")
	})
}
