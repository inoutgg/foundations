package must

// Must1 panics if err is not nil.
func Must1(err error) {
	if err != nil {
		panic(err)
	}
}

// Must panics if err is not nil and otherwise returns value.
//
// It is named as Must without a number suffix cause functions with
// two parameters are most common.
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

// Must3 panics if err is not nil, otherwise returns value1 and value2.
func Must3[T1 any, T2 any](value1 T1, value2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}

	return value1, value2
}
