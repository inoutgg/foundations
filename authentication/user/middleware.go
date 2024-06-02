package user

import (
	"context"
	"net/http"

	"log/slog"

	"go.inout.gg/common/authentication/strategy"
	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/http/errorhandler"
	"go.inout.gg/common/http/htmx"
	"go.inout.gg/common/http/middleware"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// Config is the configuration for the middleware.
type Config struct {
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
	config *Config,
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
			}

			newCtx := context.WithValue(r.Context(), kCtxKey, user)
			next.ServeHTTP(
				w,
				r.WithContext(newCtx),
			)
		})
	}
}

// PreventAuthenticatedUserAccessMiddleware is a middleware that redirects the user to the
// provided URL if the user is authenticated.
//
// Make sure to use the Middleware before adding this one.
func PreventAuthenticatedUserAccessMiddleware(redirectUrl string) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsAuthorized(r.Context()) {
				htmx.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// FromRequest returns the user from the request context if it exists.
//
// Make sure to use the Middleware before calling this function.
func FromRequest[T any](r *http.Request) *strategy.Session[T] {
	return FromContext[T](r.Context())
}

// FromContext returns the user from the context if it exists.
//
// Make sure to use the Middleware before calling this function.
func FromContext[T any](ctx context.Context) *strategy.Session[T] {
	if user, ok := ctx.Value(kCtxKey).(*strategy.Session[T]); ok {
		return user
	}

	return nil
}

// IsAuthorized returns true if the user is authorized.
//
// It is a shortcut for FromContext(r.Context())!=nil.
func IsAuthorized(ctx context.Context) bool {
	return FromContext[any](ctx) != nil
}
