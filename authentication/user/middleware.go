package user

import (
	"context"
	"net/http"

	"log/slog"

	"go.inout.gg/common/authentication/strategy"
	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/http/errorhandler"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// MiddlewareConfig is the configuration for the middleware.
type MiddlewareConfig struct {
	Logger *slog.Logger

	// ErrorHandler is the error handler that is called when the user is not
	// authenticated.
	// If nil, the default error handler is used.
	ErrorHandler errorhandler.ErrorHandler

	// Passthrough controls whether the request should be failed
	// on unauthorized access.
	Passthrough bool
}

// Middleware returns a middleware that authenticates the user and
// adds it to the request context.
//
// If the user is not authenticated, the error handler is called.
func Middleware[T any](
	authenticator strategy.Authenticator[T],
	config *MiddlewareConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		eh := config.ErrorHandler
		if eh == nil {
			eh = errorhandler.DefaultErrorHandler
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticator.Authenticate(w, r)
			if err != nil {
				// If Passthrough is not set ignore the error and continue.
				if !config.Passthrough {
					eh.ServeHTTP(
						w,
						r,
						httperror.FromError(err, http.StatusUnauthorized, "unauthorized access"),
					)

					return
				}

				if config.Logger != nil {
					config.Logger.WarnContext(r.Context(), "unauthorized access", "error", err)
				}
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
