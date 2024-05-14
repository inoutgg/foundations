package driverpgxv5

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/common/authentication/internal/query"
)

var _ Querier = (*Driver)(nil)
var _ Querier = (*ExecutorTx)(nil)

type Querier interface {
	Queries() *query.Queries
}

// Driver is a pgx/v5 database driver for use with the authentication package.
type Driver struct {
	pool    *pgxpool.Pool
	logger  *slog.Logger
	queries *query.Queries
}

// New returns a new pgx/v5 database driver for use with the authentication package.
//
// It takes a pgxpool.Pool for use with the driver. The pool should be open
// while the driver is in use.
func New(logger *slog.Logger, pool *pgxpool.Pool) *Driver {
	return &Driver{
		logger:  logger,
		pool:    pool,
		queries: query.New(pool),
	}
}

func (d *Driver) Queries() *query.Queries { return d.queries }

type ExecutorTx struct {
	queries *query.Queries
	tx      pgx.Tx
}

func (t *ExecutorTx) Queries() *query.Queries            { return t.queries }
func (t *ExecutorTx) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *ExecutorTx) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }

func (d *Driver) Begin(ctx context.Context) (*ExecutorTx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication/db: failed to begin transaction: %w", err)
	}

	return &ExecutorTx{
		queries: d.queries.WithTx(tx),
		tx:      tx,
	}, nil
}
