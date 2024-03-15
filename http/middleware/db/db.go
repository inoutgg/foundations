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
	ErrDBPoolNotFound = errors.New("http/db: unable to retrieve db pool from request context. Make sure to use corresponding middleware.")
)

// From returns the pool associated with the given http request.
func From(req *http.Request) (*pgxpool.Pool, error) {
	ctx := req.Context()
	if pool, ok := ctx.Value(kCtxKey).(*pgxpool.Pool); ok {
		return pool, nil
	}

	return nil, ErrDBPoolNotFound
}

func Middleware(db *pgxpool.Pool) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), kCtxKey, db)))
		})
	}
}
