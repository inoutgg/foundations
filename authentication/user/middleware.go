package user

import (
	"context"
	"net/http"

	httperror "github.com/atcirclesquare/common/http/error"
	"github.com/atcirclesquare/common/http/errorhandler"
	"github.com/google/uuid"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

type User[T any] struct {
	ID uuid.UUID
	T  T
}

// Authenticator authenticates the user.
type Authenticator[T any] interface {
	Authenticate(ctx context.Context, r *http.Request) (*User[T], error)
}

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
