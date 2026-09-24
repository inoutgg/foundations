//go:build otelc

package dbsql

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"go.inout.gg/foundations/dbsql/sqlcomment"
)

type queryRecorder struct {
	queries []string
}

func (r *queryRecorder) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	r.queries = append(r.queries, data.SQL)
	return ctx
}

func (r *queryRecorder) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (r *queryRecorder) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	return ctx
}

func (r *queryRecorder) TraceBatchQuery(_ context.Context, _ *pgx.Conn, data pgx.TraceBatchQueryData) {
	r.queries = append(r.queries, data.SQL)
}

func (r *queryRecorder) TraceBatchEnd(context.Context, *pgx.Conn, pgx.TraceBatchEndData) {}

func (r *queryRecorder) last(t *testing.T) string {
	t.Helper()
	require.NotEmpty(t, r.queries)
	return r.queries[len(r.queries)-1]
}

func newTracedPool(t *testing.T) (*pgxpool.Pool, *queryRecorder) {
	t.Helper()

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://test:test@localhost:5432/test"
	}

	recorder := &queryRecorder{}
	pool, err := NewPool(t.Context(), connString, func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = recorder
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(t.Context()))
	return pool, recorder
}

func TestInstrumentedSQLComments(t *testing.T) {
	pool, recorder := newTracedPool(t)
	ctx := sqlcomment.WithTag(context.Background(), sqlcomment.KeyApplication, "api")
	want := "select 1 /*" + (sqlcomment.Tags{
		sqlcomment.KeyDBDriver:    sqlcomment.NewDriverVersionTagger().Tag(ctx)[sqlcomment.KeyDBDriver],
		sqlcomment.KeyApplication: "api",
	}).Marshal() + "*/"

	_, err := pool.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))

	var result int
	require.NoError(t, pool.QueryRow(ctx, "select 1").Scan(&result))
	require.Equal(t, want, recorder.last(t))

	batch := &pgx.Batch{}
	batch.Queue("select 1")
	require.NoError(t, pool.SendBatch(ctx, batch).Close())
	require.Equal(t, want, recorder.last(t))
	require.Equal(t, "select 1", batch.QueuedQueries[0].SQL)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck
	_, err = tx.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))
	txBatch := &pgx.Batch{}
	txBatch.Queue("select 1")
	require.NoError(t, tx.SendBatch(ctx, txBatch).Close())
	require.Equal(t, want, recorder.last(t))
	require.Equal(t, "select 1", txBatch.QueuedQueries[0].SQL)

	nested, err := tx.Begin(ctx)
	require.NoError(t, err)
	defer nested.Rollback(ctx) //nolint:errcheck
	require.NoError(t, nested.QueryRow(ctx, "select 1").Scan(&result))
	require.Equal(t, want, recorder.last(t))

	ctx = sqlcomment.Skip(ctx)
	_, err = nested.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, "select 1", recorder.last(t))
}

func TestInstrumentedTraceContext(t *testing.T) {
	old := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(old)
	pool, recorder := newTracedPool(t)

	ctx := propagation.TraceContext{}.Extract(context.Background(), propagation.MapCarrier{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	})
	_, err := pool.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Contains(t, recorder.last(t), "traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01'")
}

func TestInstrumentedManualWrapper(t *testing.T) {
	pool, recorder := newTracedPool(t)
	wrapped := sqlcomment.NewWithDefaults(pool,
		sqlcomment.WithTags(sqlcomment.Tags{sqlcomment.KeyApplication: "manual"}),
		sqlcomment.WithDriverVerTag(),
	)
	ctx := sqlcomment.WithTag(context.Background(), "request id", "a b")
	want := "select 1 /*" + (sqlcomment.Tags{
		sqlcomment.KeyDBDriver:    sqlcomment.NewDriverVersionTagger().Tag(ctx)[sqlcomment.KeyDBDriver],
		sqlcomment.KeyApplication: "manual",
		"request id":              "a b",
	}).Marshal() + "*/"

	_, err := wrapped.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))

	var value int
	require.NoError(t, wrapped.QueryRow(ctx, "select 1").Scan(&value))
	require.Equal(t, want, recorder.last(t))

	batch := &pgx.Batch{}
	batch.Queue("select 1")
	require.NoError(t, wrapped.SendBatch(ctx, batch).Close())
	require.Equal(t, want, recorder.last(t))
	require.Equal(t, "select 1", batch.QueuedQueries[0].SQL)

	tx, err := wrapped.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck
	_, err = tx.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))

	_, err = pool.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Contains(t, recorder.last(t), "request+id='a%20b'")
}
