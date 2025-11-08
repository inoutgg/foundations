package dbsql

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ DB = (*pgx.Conn)(nil)
var _ DB = (*pgxpool.Pool)(nil)
var _ DBTX = (*pgx.Conn)(nil)
var _ DBTX = (*pgxpool.Pool)(nil)

// DBTX is an interface common to pgx.Conn, pgx.Tx and pgxpool.Pool.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Begin(context.Context) (pgx.Tx, error)
}

// DB is an interface common to pgx.Conn and pgxpool.Pool.
type DB interface {
	DBTX
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
