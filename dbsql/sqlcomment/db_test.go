package sqlcomment

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type queryRecorder struct {
	queries []string
}

func (r *queryRecorder) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
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

func newManualDB(t *testing.T, options ...Option) (*DB, *queryRecorder) {
	t.Helper()

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = os.Getenv("PGX_TEST_DATABASE")
	}

	if connString == "" {
		connString = "postgres://test:test@localhost:5432/test"
	}

	recorder := &queryRecorder{}
	cfg, err := pgxpool.ParseConfig(connString)
	require.NoError(t, err)

	cfg.ConnConfig.Tracer = recorder
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(t.Context()))

	return NewWithDefaults(pool, options...), recorder
}

func TestManualDBComments(t *testing.T) {
	wrapped, recorder := newManualDB(t, WithTags(Tags{KeyApplication: "api"}))
	ctx := WithTag(context.Background(), "request id", "a b")
	want := "select 1 /*application='api',request+id='a%20b'*/"

	_, err := wrapped.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))

	_, err = wrapped.Exec(WithTag(ctx, KeyApplication, "context"), "select 1")
	require.NoError(t, err)
	require.Equal(t, "select 1 /*application='context',request+id='a%20b'*/", recorder.last(t))

	var value int
	require.NoError(t, wrapped.QueryRow(ctx, "select 1").Scan(&value))
	require.Equal(t, want, recorder.last(t))

	batch := &pgx.Batch{}
	batch.Queue("select 1")
	require.NoError(t, wrapped.SendBatch(ctx, batch).Close())
	require.Equal(t, want, recorder.last(t))
	require.Equal(t, "select 1", batch.QueuedQueries[0].SQL)

	tx, err := wrapped.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	defer func() { require.NoError(t, tx.Rollback(ctx)) }()

	_, err = tx.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))

	nested, err := tx.Begin(ctx)
	require.NoError(t, err)

	defer func() { require.NoError(t, nested.Rollback(ctx)) }()

	require.NoError(t, nested.QueryRow(ctx, "select 1").Scan(&value))
	require.Equal(t, want, recorder.last(t))

	_, err = nested.Exec(Skip(ctx), "select 1")
	require.NoError(t, err)
	require.Equal(t, "select 1", recorder.last(t))

	_, err = nested.Exec(ctx, "select 1")
	require.NoError(t, err)
	require.Equal(t, want, recorder.last(t))
}
