package cookie

import (
	"context"
	"errors"
	"net/http"

	"github.com/atcirclesquare/common/http/middleware"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

// Config is the configuration for the cookie middleware.
type Config struct {
	// Prod indicates whether the middleware is running in production mode.
	//
	// If the middleware is running in production mode, it will set the secure flag
	// on the cookies.
	Prod bool
}

// Middleware returns a middleware that adds cookie manager to the request context.
func Middleware(config Config) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(
				context.WithValue(r.Context(), kCtxKey, cookie{
					req:    r,
					resp:   w,
					secure: config.Prod,
				})),
			)
		})
	}
}

// FromRequest returns cookie manager if one is available.
func FromRequest(req *http.Request) (Cookie, error) {
	cookie, ok := req.Context().Value(kCtxKey).(Cookie)
	if !ok {
		return nil, errors.New(
			"http/cookie: unable to retrieve request context. Make sure to use corresponding middleware.",
		)
	}

	return cookie, nil
}
