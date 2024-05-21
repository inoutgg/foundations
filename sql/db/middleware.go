package db

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/common/http/middleware"
)

// FromRequest returns the pool associated with the given http request.
func FromRequest(req *http.Request) (*pgxpool.Pool, error) {
	return FromContext(req.Context())
}

// Middleware returns a middleware that injects the given pool into the request context.
func Middleware(db *pgxpool.Pool) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(WithContext(req.Context(), db)))
		})
	}
}
