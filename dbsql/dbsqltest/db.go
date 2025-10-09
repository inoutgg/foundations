package dbsqltest

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX represents a common interface for a database connection,.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

// Migrator applies the migration to the database.
type Migrator interface {
	// Up applies the migration to the database.
	Up(context.Context, *pgx.Conn) error

	// Hash returns a unique identifier for the migrations to be applied.
	//
	// Hash is used to uniquely identify database template in the target
	// database.
	Hash() string
}
