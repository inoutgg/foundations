package pgxhook

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otelc/pkg/hook"

	"go.inout.gg/foundations/dbsql/sqlcomment"
)

// Pool, pool.Conn and transaction methods all delegate to pgx.Conn. Hooking
// the connection methods comments each query once without wrapping the pool.
const (
	queryParam = 2 // receiver, context, SQL
	batchParam = 2 // receiver, context, batch
)

// BeforeExec adds a sqlcommenter comment before pgx executes a statement.
//
//nolint:revive // The hook signature must match pgx.Conn.Exec with otelc's hook context first.
func BeforeExec(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, sql string, _ ...any) {
	ictx.SetParam(queryParam, sqlcomment.Append(ctx, sql))
}

// BeforeQuery covers Query and QueryRow (which delegates to Query).
//
//nolint:revive // The hook signature must match pgx.Conn.Query with otelc's hook context first.
func BeforeQuery(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, sql string, _ ...any) {
	ictx.SetParam(queryParam, sqlcomment.Append(ctx, sql))
}

// BeforeSendBatch copies queued queries before pgx mutates them on execution.
//
//nolint:revive // The hook signature must match pgx.Conn.SendBatch with otelc's hook context first.
func BeforeSendBatch(ictx hook.HookContext, _ *pgx.Conn, ctx context.Context, batch *pgx.Batch) {
	ictx.SetParam(batchParam, sqlcomment.CommentBatch(ctx, batch))
}
