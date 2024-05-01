package db

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolOption sets options for the database pool.
//
// It is convinient way to set options for the database pool.
type PoolOption interface {
	Apply(*pgxpool.Config)
}

type PoolOptionFunc func(c *pgxpool.Config)

func (o PoolOptionFunc) Apply(c *pgxpool.Config) { o(c) }

// WithTracer sets the query tracer for the database pool.
//
// Checkout [pgxotel.Tracer] for an opentelemetry tracing implementation.
func WithTracer(t pgx.QueryTracer) PoolOption {
	return PoolOptionFunc(func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = t
	})
}
