package pgxhook

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otelc/pkg/hook"
	"go.opentelemetry.io/otelc/pkg/hook/hooktest"

	"go.inout.gg/foundations/dbsql/sqlcomment"
)

func TestQueryHooks(t *testing.T) {
	t.Parallel()

	ctx := sqlcomment.WithTag(context.Background(), "request id", "a b")
	want := "select 1 /*" + (sqlcomment.Tags{
		sqlcomment.KeyDBDriver: sqlcomment.NewDriverVersionTagger().Tag(ctx)[sqlcomment.KeyDBDriver],
		"request id":           "a b",
	}).Marshal() + "*/"

	for _, hook := range []struct {
		call func(hook.HookContext, *pgx.Conn, context.Context, string, ...any)
		name string
	}{
		{name: "Exec", call: BeforeExec},
		{name: "Query", call: BeforeQuery},
	} {
		t.Run(hook.name, func(t *testing.T) {
			t.Parallel()

			ictx := hooktest.NewMockHookContext((*pgx.Conn)(nil), ctx, "select 1")
			hook.call(ictx, nil, ctx, "select 1")
			require.Equal(t, want, ictx.GetParam(queryParam))
		})
	}
}

func TestHooksSkip(t *testing.T) {
	t.Parallel()

	ctx := sqlcomment.Skip(context.Background())
	ictx := hooktest.NewMockHookContext((*pgx.Conn)(nil), ctx, "select 1")
	BeforeExec(ictx, nil, ctx, "select 1")
	require.Equal(t, "select 1", ictx.GetParam(queryParam))

	batch := &pgx.Batch{}
	batch.Queue("select 1")
	ictx = hooktest.NewMockHookContext((*pgx.Conn)(nil), ctx, batch)
	BeforeSendBatch(ictx, nil, ctx, batch)
	require.Same(t, batch, ictx.GetParam(batchParam))
}

func TestBeforeSendBatch(t *testing.T) {
	t.Parallel()

	ctx := sqlcomment.WithTag(context.Background(), sqlcomment.KeyApplication, "api")
	batch := &pgx.Batch{}
	batch.Queue("select 1")
	ictx := hooktest.NewMockHookContext((*pgx.Conn)(nil), ctx, batch)

	BeforeSendBatch(ictx, nil, ctx, batch)

	commented, ok := ictx.GetParam(batchParam).(*pgx.Batch)
	require.True(t, ok)
	require.NotSame(t, batch, commented)
	require.NotSame(t, batch.QueuedQueries[0], commented.QueuedQueries[0])
	require.Equal(t, "select 1", batch.QueuedQueries[0].SQL)
	require.Contains(t, commented.QueuedQueries[0].SQL, "application='api'")
}
