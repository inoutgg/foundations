package errorhandler

import (
	"net/http"
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
	Handle(w http.ResponseWriter, r *http.Request, err error)
}

// ErrorHandlerFunc is an adapter to handle errors returned by an Handler.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

func (f ErrorHandlerFunc) Handle(w http.ResponseWriter, r *http.Request, err error) {
	f(w, r, err)
}

// WithError returns an http.Handler that handles errors by onError.
func WithError(errorHandler ErrorHandler) func(Handler) http.Handler {
	return func(next Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := next.ServeHTTP(w, r)
			if err != nil {
				errorHandler.Handle(w, r, err)
			}
		})
	}
}
