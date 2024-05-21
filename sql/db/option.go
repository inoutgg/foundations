package db

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTracer sets the query tracer for the database pool.
//
// Checkout [pgxotel.Tracer] for an opentelemetry tracing implementation.
func WithTracer(t pgx.QueryTracer) func(c *pgxpool.Config) {
	return func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = t
	}
}
