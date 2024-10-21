package pointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToValue(t *testing.T) {
	t.Run("non-nil pointer", func(t *testing.T) {
		value := 42
		ptr := &value
		defaultValue := 0

		result := ToValue(ptr, defaultValue)

		assert.Equal(t, value, result, "ToValue should return the pointed-to value for non-nil pointers")
	})

	t.Run("nil pointer", func(t *testing.T) {
		var ptr *int
		defaultValue := 10

		result := ToValue(ptr, defaultValue)

		assert.Equal(t, defaultValue, result, "ToValue should return the default value for nil pointers")
	})
}

func TestFromValue(t *testing.T) {
	t.Run("integer value", func(t *testing.T) {
		value := 42

		result := FromValue(value)

		assert.NotNil(t, result, "FromValue should return a non-nil pointer")
		assert.Equal(t, value, *result, "FromValue should return a pointer to the correct value")
	})

	t.Run("string value", func(t *testing.T) {
		value := "hello"

		result := FromValue(value)

		assert.NotNil(t, result, "FromValue should return a non-nil pointer")
		assert.Equal(t, value, *result, "FromValue should return a pointer to the correct value")
	})

	t.Run("struct value", func(t *testing.T) {
		type TestStruct struct {
			Field string
		}
		value := TestStruct{Field: "test"}

		result := FromValue(value)

		assert.NotNil(t, result, "FromValue should return a non-nil pointer")
		assert.Equal(t, value, *result, "FromValue should return a pointer to the correct value")
	})
}
