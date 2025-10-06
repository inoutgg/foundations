package dbsqltest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestTxFactory is responsible for creating transactions for testing purposes.
//
// The transaction testing pattern is powerful mechanism for testing database
// data. It spins up a transaction for each test case allowing to test database
// data in isolation concurrently. Once the test completes, the transaction is
// rolled back and the database state is reset to its initial state.
type TestTxFactory struct {
	db DBTX
}

// NewTestTxFactory creates a new TestTxFactory instance.
//
// The provided database connection pool must be open and ready to use.
func NewTestTxFactory(db DBTX) TestTxFactory {
	return TestTxFactory{db: db}
}

// NewTestTxFactoryFromConnString creates a new connection pool from the provided connection string and
// returns a new TestTxFactory instance.
//
// A TestTxFactory instance is returned along with a cleanup function that
// should be called after the test suite completes.
func NewTestTxFactoryFromConnString(ctx context.Context, connString string) (TestTxFactory, func(), error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return TestTxFactory{}, nil, err
	}

	return NewTestTxFactory(pool), pool.Close, nil
}

// Tx spawns a database transaction for a given test. Once the test completes,
// the transaction is automatically rolled back on cleanup and the database
// state is reset to its initial state.
//
// If it fails to start a new transaction it will panic.
func (f TestTxFactory) Tx(tb testing.TB) pgx.Tx {
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
