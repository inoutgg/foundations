package sqldb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/must"
	"go.inout.gg/foundations/sqldb/internal/pgxuuid"
)

// WithTracer sets the query tracer for the database pool.
func WithTracer(t pgx.QueryTracer) func(c *pgxpool.Config) {
	return func(c *pgxpool.Config) { c.ConnConfig.Tracer = t }
}

// WithUUID adds native support for converting between Postgres UUID and google/uuid.
func WithUUID() func(c *pgxpool.Config) {
	return func(c *pgxpool.Config) {
		origAfterConnect := c.AfterConnect
		c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			pgxuuid.Register(conn.TypeMap())
			if origAfterConnect != nil {
				return origAfterConnect(ctx, conn)
			}
			return nil
		}
	}
}

// MustPool creates a new connection pool and panics on error.
func MustPool(ctx context.Context, connString string, cfgs ...func(*pgxpool.Config)) *pgxpool.Pool {
	return must.Must(NewPool(ctx, connString, cfgs...))
}

// NewPool creates a new connection pool using the provided connection string.
//
// Optional cfgs like WithUUID or WithTracer can be provided.
func NewPool(ctx context.Context, connString string, cfgs ...func(*pgxpool.Config)) (pool *pgxpool.Pool, err error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("foundations/sqldb: failed to parse database connection string: %w", err)
	}
	for _, f := range cfgs {
		f(config)
	}

	pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("foundations/sqldb: failed to create a new database pool: %w", err)
	}
	defer func() {
		if err != nil {
			if pool != nil {
				pool.Close()
			}
		}
	}()

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("foundations/sqldb: failed to connect to the database at %s: %w", connString, err)
	}

	return pool, nil
}
