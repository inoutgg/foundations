package httpmiddleware

import "net/http"

var _ Middleware = (MiddlewareFunc)(nil)

// Middleware is a type that wraps an http.Handler with additional logic.
type Middleware interface {
	// Middleware wraps the given handler with the middleware.
	Middleware(http.Handler) http.Handler
}

// MiddlewareFunc is a function that implements the Middleware interface.
type MiddlewareFunc func(http.Handler) http.Handler

func (m MiddlewareFunc) Middleware(h http.Handler) http.Handler {
	return m(h)
}
