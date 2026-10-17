package pgxhook

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otelc/pkg/hook"

	"go.inout.gg/foundations/dbsql/sqlcomment"
)

// Pool, pool.Conn and transaction methods delegate to pgx.Conn. These hooks
// comment each query once without wrapping the pool.
const (
	contextParam = 1 // receiver, context
	queryParam   = 2 // receiver, context, SQL
	batchParam   = 2 // receiver, context, batch
)

// BeforeExec binds the default name extractor and comments the SQL statement.
//
//nolint:revive // The hook signature must match pgx.Conn.Exec with otelc's hook context first.
func BeforeExec(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, sql string, _ ...any) {
	if _, ok := ctx.Value(queryNameExtractorKey{}).(QueryNameExtractor); !ok {
		ctx = WithQueryNameExtractor(ctx, DefaultQueryNameExtractor)
	}

	ictx.SetParam(contextParam, ctx)
	ictx.SetParam(queryParam, sqlcomment.Append(ctx, sql))
}

// BeforeQuery binds the default name extractor and comments Query and QueryRow.
//
//nolint:revive // The hook signature must match pgx.Conn.Query with otelc's hook context first.
func BeforeQuery(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, sql string, _ ...any) {
	if _, ok := ctx.Value(queryNameExtractorKey{}).(QueryNameExtractor); !ok {
		ctx = WithQueryNameExtractor(ctx, DefaultQueryNameExtractor)
	}

	ictx.SetParam(contextParam, ctx)
	ictx.SetParam(queryParam, sqlcomment.Append(ctx, sql))
}

// BeforeSendBatch binds the default extractor and copies queries before pgx mutates them.
//
//nolint:revive // The hook signature must match pgx.Conn.SendBatch with otelc's hook context first.
func BeforeSendBatch(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, batch *pgx.Batch) {
	if _, ok := ctx.Value(queryNameExtractorKey{}).(QueryNameExtractor); !ok {
		ctx = WithQueryNameExtractor(ctx, DefaultQueryNameExtractor)
	}

	ictx.SetParam(contextParam, ctx)
	ictx.SetParam(batchParam, sqlcomment.CommentBatch(ctx, batch))
}
