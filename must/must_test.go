package must

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMust1(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		assert.NotPanics(t, func() { Must1(nil) })
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must1(errors.New("panic")) })
	})
}

func TestMust(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		var v1 int
		assert.NotPanics(t, func() { v1 = Must(123, nil) })
		assert.Equal(t, v1, 123)
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must(123, errors.New("panic")) })
	})
}

func TestMust3(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		var v1 int
		var v2 string
		assert.NotPanics(t, func() { v1, v2 = Must3(123, "my string", nil) })
		assert.Equal(t, v1, 123)
		assert.Equal(t, v2, "my string")
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must3(123, "my string", errors.New("panic")) })
	})
}

func TestMust4(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		var v1 int
		var v2 string
		var v3 bool
		assert.NotPanics(t, func() { v1, v2, v3 = Must4(123, "my string", false, nil) })
		assert.Equal(t, v1, 123)
		assert.Equal(t, v2, "my string")
		assert.Equal(t, v3, false)
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must4(123, "my string", false, errors.New("panic")) })
	})
}
