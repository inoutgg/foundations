package error

import "strings"

// Check HttpError compliance to error.
var _ error = (*HttpError)(nil)

// HttpError is an error that cares additional information about the HTTP status for the error.
type HttpError struct {
	message string
	code    int
	errors  []error
}

// New creates a new HttpError with the given message and code.
func New(message string, code int, errors ...error) HttpError {
	return HttpError{message, code, errors}
}

// FromError creates a new HttpError with the given error and code.
func FromError(err error, code int, message ...string) HttpError {
	return New(strings.Join(message, " "), code, err)
}

func (e HttpError) Unwrap() []error { return e.errors }
func (e HttpError) Error() string   { return e.message }
func (e HttpError) StatusCode() int { return e.code }
