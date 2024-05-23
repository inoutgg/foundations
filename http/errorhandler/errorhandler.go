package errorhandler

import (
	"net/http"

	httperror "go.inout.gg/common/http/error"
)

var _ Handler = (HandlerFunc)(nil)
var _ ErrorHandler = (ErrorHandlerFunc)(nil)

// Handler is like http.Handler, but with an additional error return value.
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

// HandlerFunc is an adapter to allow the use of ordinary functions as HTTP handlers.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

// ErrorHandler is an interface that can handle errors returned by an Handler.
type ErrorHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, err error)
}

// ErrorHandlerFunc is an adapter to handle errors returned by an Handler.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

func (f ErrorHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, err error) {
	f(w, r, err)
}

// WithErrorHandler returns an http.Handler that handles errors by errorHandler.
func WithErrorHandler(errorHandler ErrorHandler) func(HandlerFunc) http.HandlerFunc {
	return func(next HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := next.ServeHTTP(w, r)
			if err != nil {
				errorHandler.ServeHTTP(w, r, err)
			}
		})
	}
}

// DefaultErrorHandler is the default error handler.
var DefaultErrorHandler = ErrorHandlerFunc(func(w http.ResponseWriter, r *http.Request, err error) {
	if err, ok := err.(httperror.HttpError); ok {
		http.Error(w, err.Error(), err.StatusCode())
		return
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
})
