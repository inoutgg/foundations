// Package must provides a convenient way to panic on errors.
package must

// Must1 panics if err is not nil.
func Must1(err error) {
	if err != nil {
		panic(err)
	}
}

// Must panics if err is not nil and otherwise returns v.
//
// It is named as Must without a number suffix cause functions with
// two parameters are most common.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// Must3 panics if err is not nil, otherwise returns v1 and v2.
func Must3[T1 any, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}

	return v1, v2
}

// Must4 panics if err is not nil, otherwise returns v1, v2 and v3.
func Must4[T1 any, T2 any, T3 any](v1 T1, v2 T2, v3 T3, err error) (T1, T2, T3) {
	if err != nil {
		panic(err)
	}

	return v1, v2, v3
}
