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
		assert.Equal(t, 123, v1)
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must(123, errors.New("panic")) })
	})
}

func TestMust3(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		var (
			v1 int
			v2 string
		)

		assert.NotPanics(t, func() { v1, v2 = Must3(123, "my string", nil) })
		assert.Equal(t, 123, v1)
		assert.Equal(t, "my string", v2)
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must3(123, "my string", errors.New("panic")) })
	})
}

func TestMust4(t *testing.T) {
	t.Run("should return normally", func(t *testing.T) {
		var (
			v1 int    //nolint:varnamelen // test
			v2 string //nolint:varnamelen // test
			v3 bool   //nolint:varnamelen // test
		)

		assert.NotPanics(t, func() { v1, v2, v3 = Must4(123, "my string", false, nil) })
		assert.Equal(t, 123, v1)
		assert.Equal(t, "my string", v2)
		assert.False(t, v3)
	})
	t.Run("should panic on error", func(t *testing.T) {
		assert.Panics(t, func() { Must4(123, "my string", false, errors.New("panic")) })
	})
}
