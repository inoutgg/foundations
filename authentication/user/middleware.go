package user

import (
	"context"
	"net/http"

	"go.inout.gg/common/authentication/strategy"
	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/http/errorhandler"
	"log/slog"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// MiddlewareConfig is the configuration for the middleware.
type MiddlewareConfig[T any] struct {
	RaiseOnUnauthorizedAccess bool
	Authenticator             strategy.Authenticator[T]
	ErrorHandler              errorhandler.ErrorHandler
	Logger                    *slog.Logger
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
				if config.RaiseOnUnauthorizedAccess {
					config.ErrorHandler.ServeHTTP(
						w,
						r,
						httperror.FromError(err, http.StatusUnauthorized, "unauthorized access"),
					)

					return
				}

				config.Logger.InfoContext(r.Context(), "unauthorized access", "error", err)
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
func FromRequest[T any](r *http.Request) *strategy.User[T] {
	return FromContext[T](r.Context())
}

// FromContext returns the user from the context if it exists.
//
// Make sure to use the Middleware before calling this function.
func FromContext[T any](ctx context.Context) *strategy.User[T] {
	if user, ok := ctx.Value(kCtxKey).(*strategy.User[T]); ok {
		return user
	}

	return nil
}
