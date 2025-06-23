package dbsqltest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	CopyFrom(
		ctx context.Context,
		tableName pgx.Identifier,
		columnNames []string,
		rowSrc pgx.CopyFromSource,
	) (int64, error)
}

func WithTx(t *testing.T, fn func(DBTX) error) {
	t.Helper()

	ctx := t.Context()
	pool, err := NewDB(ctx, nil)
	require.NoError(t, err)

	tx, err := pool.Pool().Begin(ctx)
	require.NoError(t, err)

	err = fn(tx)
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback(ctx) })
}
