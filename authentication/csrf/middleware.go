// Package csrf implements a CSRF protection middleware based on the double
// submit cookie pattern.
package csrf

import (
	"context"
	"errors"
	"net/http"
	"slices"

	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/http/errorhandler"
	"go.inout.gg/common/http/middleware"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// MiddlewareConfig is the configuration for the CSRF middleware.
type MiddlewareConfig struct {
	TokenOption    *TokenOption
	IgnoredMethods []string                  // optional (default: [GET, HEAD, OPTIONS, TRACE])
	ErrorHandler   errorhandler.ErrorHandler // optional (default: errorhandler.DefaultErrorHandler)
}

// WithTokenOption sets the TokenOption on the middleware config.
func WithTokenOption(opt *TokenOption) func(*MiddlewareConfig) {
	return func(cfg *MiddlewareConfig) { cfg.TokenOption = opt }
}

// Middleware returns a middleware that adds CSRF token to the request context.
func Middleware(config ...func(*MiddlewareConfig)) middleware.MiddlewareFunc {
	cfg := MiddlewareConfig{
		IgnoredMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
			http.MethodTrace,
		},
		ErrorHandler: errorhandler.DefaultErrorHandler,
	}
	for _, f := range config {
		f(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, err := newToken(cfg.TokenOption)
			if err != nil {
				cfg.ErrorHandler.ServeHTTP(w, r, err)
				return
			}

			newCtx := context.WithValue(r.Context(), kCtxKey, tok)

			if slices.Contains(cfg.IgnoredMethods, r.Method) {
				next.ServeHTTP(w, r.WithContext(newCtx))
				return
			}

			err = validateRequest(r, cfg.TokenOption)
			if err != nil {
				err := httperror.FromError(err, http.StatusForbidden, "invalid CSRF token")
				cfg.ErrorHandler.ServeHTTP(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

// FromRequest returns the CSRF token associated with the given HTTP request.
func FromRequest(r *http.Request) (*Token, error) {
	return FromContext(r.Context())
}

// FromContext returns the CSRF token associated with the given context.
func FromContext(ctx context.Context) (*Token, error) {
	tok, ok := ctx.Value(kCtxKey).(*Token)
	if !ok {
		return nil, errors.New(
			"authentication/csrf: unable to retrieve request context. Make sure to use corresponding middleware.",
		)
	}

	return tok, nil
}
