package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

var (
	ErrDBPoolNotFound = errors.New("sql/db: unable to retrieve db pool from request context.")
)

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
