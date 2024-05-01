package db

import (
	"context"
	"errors"
	"net/http"

	"github.com/atcirclesquare/common/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

var (
	ErrDBPoolNotFound = errors.New(
		"sql/db: unable to retrieve db pool from request context. Make sure to use corresponding middleware.",
	)
)

// FromRequest returns the pool associated with the given http request.
func FromRequest(req *http.Request) (*pgxpool.Pool, error) {
	ctx := req.Context()
	if pool, ok := ctx.Value(kCtxKey).(*pgxpool.Pool); ok {
		return pool, nil
	}

	return nil, ErrDBPoolNotFound
}

// Middleware returns a middleware that injects the given pool into the request context.
func Middleware(db *pgxpool.Pool) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), kCtxKey, db)))
		})
	}
}
