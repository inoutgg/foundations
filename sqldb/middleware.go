package sqldb

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/http/httpmiddleware"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

var ErrDBPoolNotFound = errors.New("foundations/sqldb: failed to retrieve db pool from context.")

// WithContext returns a new context with the given pool.
func WithContext(ctx context.Context, pool *pgxpool.Pool) context.Context {
	return context.WithValue(ctx, kCtxKey, pool)
}

// FromContext returns the pool associated with the given context.
func FromContext(ctx context.Context) (*pgxpool.Pool, error) {
	if pool, ok := ctx.Value(kCtxKey).(*pgxpool.Pool); ok {
		return pool, nil
	}

	return nil, ErrDBPoolNotFound
}

// FromRequest returns the pool associated with the given http request.
func FromRequest(req *http.Request) (*pgxpool.Pool, error) {
	return FromContext(req.Context())
}

// Middleware returns a middleware that injects the given pool into the request context.
func Middleware(db *pgxpool.Pool) httpmiddleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(WithContext(req.Context(), db)))
		})
	}
}
