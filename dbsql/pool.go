package dbsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"go.inout.gg/foundations/must"
)

// WithSearchPath sets the search path for the database pool.
func WithSearchPath(schema string) func(c *pgxpool.Config) {
	return func(c *pgxpool.Config) { c.ConnConfig.RuntimeParams["search_path"] = schema }
}

// MustPool creates a new connection pool and panics on error.
func MustPool(ctx context.Context, connString string, opts ...func(*pgxpool.Config)) *pgxpool.Pool {
	return must.Must(NewPool(ctx, connString, opts...))
}

// NewPool creates a new connection pool using the provided connection string.
func NewPool(
	ctx context.Context,
	connStr string,
	opts ...func(*pgxpool.Config),
) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf(
			"dbsql: failed to parse database connection string: %w",
			err,
		)
	}

	for _, f := range opts {
		f(cfg)
	}

	return newPool(ctx, cfg)
}

// newPool creates a new connection pool using the provided configuration.
func newPool(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	var err error

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("dbsql: failed to create a new database pool: %w", err)
	}

	defer func() {
		if err != nil && pool != nil {
			pool.Close()
		}
	}()

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"dbsql: failed to connect to the database at %s: %w",
			cfg.ConnString(),
			err,
		)
	}

	err = registerPoolMetrics(pool)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
