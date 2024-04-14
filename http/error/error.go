package error

// Check HttpError compliance to error.
var _ error = (*HttpError)(nil)

// HttpError is an error that cares additional information about the HTTP status for the error.
type HttpError struct {
	err     error
	message string
	code    int
}

// FromString creates a new HttpError with the given message and code.
func New(message string, code int) HttpError {
	return HttpError{
		message: message,
		code:    code,
	}
}

// Unwrap returns the underlying error if any.
func (e HttpError) Unwrap() error {
	return e.err
}

// FromError creates a new HttpError with the given error and code.
func FromError(err error, message string, code int) HttpError {
	return HttpError{
		err:     err,
		message: message,
		code:    code,
	}
}

func (e HttpError) StatusCode() int {
	return e.code
}

func (e HttpError) Error() string {
	return e.message
}
