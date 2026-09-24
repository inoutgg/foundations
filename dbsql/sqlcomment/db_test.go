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

func (r *queryRecorder) TraceBatchStart(
	ctx context.Context,
	_ *pgx.Conn,
	_ pgx.TraceBatchStartData,
) context.Context {
	return ctx
}

func (r *queryRecorder) TraceBatchQuery(
	_ context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchQueryData,
) {
	r.queries = append(r.queries, data.SQL)
}

func (r *queryRecorder) TraceBatchEnd(context.Context, *pgx.Conn, pgx.TraceBatchEndData) {}

func (r *queryRecorder) Reset() {
	r.queries = nil
}

func (r *queryRecorder) Last(t *testing.T) string {
	t.Helper()

	require.NotEmpty(t, r.queries, "no queries recorded")
	return r.queries[len(r.queries)-1]
}

func newTracedDB(t *testing.T, options ...Option) (*DB, *queryRecorder) {
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

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(context.Background()))
	recorder.Reset()

	return NewWithDefaults(pool, options...), recorder
}

func TestDBComments(t *testing.T) {
	t.Parallel()

	t.Run("adds SQL comment", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyApplication: "api"}))
		ctx := WithTag(context.Background(), "request id", "a b")
		want := "select 1 /*application='api',request+id='a%20b'*/"

		// Act
		_, err := wrapped.Exec(ctx, "select 1")

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
	})

	t.Run("context tags override static tags", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyRoute: "static"}))
		ctx := WithTag(context.Background(), KeyRoute, "context")
		want := "select 1 /*route='context'*/"

		// Act
		_, err := wrapped.Exec(ctx, "select 1")

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
	})

	t.Run("skip leaves SQL unchanged", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyApplication: "api"}))
		ctx := Skip(context.Background())

		// Act
		_, err := wrapped.Exec(ctx, "select 1")

		// Assert
		require.NoError(t, err)
		require.Equal(t, "select 1", recorder.Last(t))
	})

	t.Run("comments batches without mutating original", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyApplication: "api"}))
		b := &pgx.Batch{}
		b.Queue("select 1")
		want := "select 1 /*application='api'*/"

		// Act
		err := wrapped.SendBatch(context.Background(), b).Close()

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
		require.Equal(t, "select 1", b.QueuedQueries[0].SQL)
	})

	t.Run("wraps transactions", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyApplication: "api"}))
		tx, err := wrapped.Begin(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tx.Rollback(context.Background())) })
		recorder.Reset()
		want := "select 1 /*application='api'*/"

		// Act
		_, err = tx.Exec(context.Background(), "select 1")

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
	})

	t.Run("comments transaction batches without mutating original", func(t *testing.T) {
		t.Parallel()

		// Arrange
		wrapped, recorder := newTracedDB(t, WithTags(Tags{KeyApplication: "api"}))
		tx, err := wrapped.Begin(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tx.Rollback(context.Background())) })
		recorder.Reset()
		b := &pgx.Batch{}
		b.Queue("select 1")
		want := "select 1 /*application='api'*/"

		// Act
		err = tx.SendBatch(context.Background(), b).Close()

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
		require.Equal(t, "select 1", b.QueuedQueries[0].SQL)
	})
}
