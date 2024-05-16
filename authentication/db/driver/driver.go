package driver

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/common/authentication/internal/query"
)

type Querier interface {
	Queries() *query.Queries
}

type ExecutorTx interface {
	Queries() *query.Queries
	Tx() pgx.Tx
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Driver interface {
	Begin(context.Context) (ExecutorTx, error)
	Querier
}
