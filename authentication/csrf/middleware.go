package csrf

import (
	"context"
	"errors"
	"net/http"
	"slices"

	httperror "github.com/atcirclesquare/common/http/error"
	"github.com/atcirclesquare/common/http/errorhandler"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// MiddlewareConfig is the configuration for the CSRF middleware.
type MiddlewareConfig struct {
	TokenOption    *TokenOption
	IgnoredMethods []string                  // optional (default: [GET, HEAD, OPTIONS, TRACE])
	ErrorHandler   errorhandler.ErrorHandler // optional (default: errorhandler.DefaultErrorHandler)
}

// Middleware returns a middleware that adds CSRF token to the request context.
func Middleware(config ...func(*MiddlewareConfig)) func(next http.Handler) http.Handler {
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

			nr := r.WithContext(context.WithValue(r.Context(), kCtxKey, tok))

			if slices.Contains(cfg.IgnoredMethods, r.Method) {
				next.ServeHTTP(w, nr)
				return
			}

			err = validateRequest(r, cfg.TokenOption)
			if err != nil {
				err := httperror.FromError(err, "invalid CSRF token", http.StatusForbidden)
				cfg.ErrorHandler.ServeHTTP(w, r, err)
				return
			}

			next.ServeHTTP(w, nr)
		})
	}
}

// FromRequest returns the CSRF token associated with the given http request.
func FromRequest(r *http.Request) (*Token, error) {
	tok, ok := r.Context().Value(kCtxKey).(*Token)
	if !ok {
		return nil, errors.New(
			"authentication/csrf: unable to retrieve request context. Make sure to use corresponding middleware.",
		)
	}

	return tok, nil
}
