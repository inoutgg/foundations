package pointer

// ToValue converts to a value, and defaults to the [defaultValue] when it is nil.
func ToValue[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}

	return *ptr
}

// FromValue converts from a value to a pointer.
func FromValue[T any](value T) *T {
	return &value
}
