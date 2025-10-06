package httperror

import (
	"strings"
)

// Check HttpError compliance to error.
var _ error = (*HTTPError)(nil)

// HTTPError is an error that cares additional information about the HTTP status for the error.
type HTTPError struct {
	message string
	errors  []error
	code    int
}

// New creates a new HttpError with the given message and code.
func New(message string, code int, errors ...error) HTTPError {
	return HTTPError{message, errors, code}
}

// FromError creates a new HttpError with the given error and code.
func FromError(err error, code int, message ...string) HTTPError {
	return New(strings.Join(message, " "), code, err)
}

func (e HTTPError) Error() string   { return e.message }
func (e HTTPError) Unwrap() []error { return e.errors }
func (e HTTPError) StatusCode() int { return e.code }
