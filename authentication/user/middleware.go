package user

import (
	"context"
	"net/http"

	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/http/errorhandler"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// MiddlewareConfig is the configuration for the middleware.
type MiddlewareConfig[T any] struct {
	Authenticator Authenticator[T]
	ErrorHandler  errorhandler.ErrorHandler
}

// Middleware returns a middleware that authenticates the user and
// adds it to the request context.
//
// If the user is not authenticated, the error handler is called.
func Middleware[T any](config *MiddlewareConfig[T]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := config.Authenticator.Authenticate(r.Context(), r)
			if err != nil {
				config.ErrorHandler.ServeHTTP(
					w,
					r,
					httperror.FromError(err, http.StatusUnauthorized, "unauthorized access"),
				)
				return
			}

			newCtx := context.WithValue(r.Context(), kCtxKey, user)
			next.ServeHTTP(
				w,
				r.WithContext(newCtx),
			)
		})
	}
}

// FromRequest returns the user from the request context if it exists.
//
// Make sure to use the Middleware before calling this function.
func FromRequest[T any](r *http.Request) *User[T] {
	if user, ok := r.Context().Value(kCtxKey).(*User[T]); ok {
		return user
	}

	return nil
}
