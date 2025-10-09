package dbsqltest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestEphemeralTx creates transactions for testing purposes.
//
// Transaction testing pattern is a powerful mechanism for testing database
// data. It spins up a transaction for each test case allowing to test database
// data in isolation concurrently. Once the test completes, the transaction is
// rolled back and the database state is reset to its initial state.
type TestEphemeralTx struct {
	db DBTX
}

// NewTestEphemeralTx creates a new TestEphemeralTx instance.
//
// The provided database connection pool must be open and ready to use.
func NewTestEphemeralTx(db DBTX) TestEphemeralTx {
	return TestEphemeralTx{db: db}
}

// NewTestEphemeralTxFromConnString creates a new connection pool from the provided connection string and
// returns a new TestEphemeralTx instance.
//
// A TestEphemeralTx instance is returned along with a cleanup function that
// should be called after the test suite completes.
func NewTestEphemeralTxFromConnString(ctx context.Context, connString string) (TestEphemeralTx, func(), error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return TestEphemeralTx{}, nil, fmt.Errorf("dbsqltest: failed to create connection pool: %w", err)
	}

	return NewTestEphemeralTx(pool), pool.Close, nil
}

// Tx spawns a database transaction for a given test. Once the test completes,
// the transaction is automatically rolled back on cleanup and the database
// state is reset to its initial state.
//
// If it fails to start a new transaction it will panic.
func (f TestEphemeralTx) Tx(tb testing.TB) pgx.Tx {
	tb.Helper()

	tx, err := f.db.Begin(tb.Context())
	require.NoError(tb, err, "failed to start transaction")

	tb.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err = tx.Rollback(ctx); err != nil {
			if !errors.Is(err, pgx.ErrTxClosed) {
				require.NoError(tb, err, "an error occurred on test cleanup")
			}
		}
	})

	return tx
}
