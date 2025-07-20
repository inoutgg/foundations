package httptimeout

import (
	"context"
	"net/http"
	"time"
)

// Middleware returns a middleware that sets a timeout for the request.
//
// The timeout is set to the duration of the request context.
func Middleware(dur time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), dur)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
