package dbsql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey struct{}

var kCtxKey = ctxKey{} //nolint:gochecknoglobals

var ErrDBPoolNotFound = errors.New("foundations/sqldb: failed to retrieve db pool from context")

// WithPool returns a new context with the given pool.
func WithPool(ctx context.Context, pool *pgxpool.Pool) context.Context {
	return context.WithValue(ctx, kCtxKey, pool)
}

// FromContext returns the pool associated with the given context.
func FromContext(ctx context.Context) (*pgxpool.Pool, error) {
	if pool, ok := ctx.Value(kCtxKey).(*pgxpool.Pool); ok {
		return pool, nil
	}

	return nil, ErrDBPoolNotFound
}
