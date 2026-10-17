//go:build otelc

package dbsql

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.inout.gg/foundations/dbsql/pgxhook"
	"go.inout.gg/foundations/dbsql/sqlcomment"
	"go.inout.gg/foundations/test/testutil"
)

func TestPool_Exec_SQLCSpanName(t *testing.T) {
	// OTel's provider is process-wide.
	oldProvider := otel.GetTracerProvider()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		require.NoError(t, provider.Shutdown(context.Background()))
	})
	ctx, parent := otel.Tracer("go.inout.gg/foundations/dbsql/pgxhook/test").Start(t.Context(), "test")
	t.Cleanup(func() { parent.End() })
	namedCtx := pgxhook.WithQueryNameExtractor(ctx, pgxhook.SQLCQueryName)

	pool, err := NewPool(t.Context(), testutil.PgxConnString(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	t.Run("should name client span after sqlc when query has an annotation", func(t *testing.T) {
		_, err := pool.Exec(namedCtx, "-- name: GetAuthor :one\nSELECT 1")
		require.NoError(t, err)

		spans := recorder.Ended()
		require.NotEmpty(t, spans)
		span := spans[len(spans)-1]
		require.Equal(t, "GetAuthor", span.Name())
		require.Equal(t, trace.SpanKindClient, span.SpanKind())
	})

	t.Run("should use SQL as span name when query has no sqlc annotation", func(t *testing.T) {
		_, err := pool.Exec(namedCtx, "SELECT 1")
		require.NoError(t, err)

		spans := recorder.Ended()
		require.Equal(t, sqlcomment.Append(namedCtx, "SELECT 1"), spans[len(spans)-1].Name())
	})

	t.Run("should end a named query span when query row is scanned", func(t *testing.T) {
		row := pool.QueryRow(namedCtx, "-- name: GetCount :one\nSELECT 1")

		var count int
		require.NoError(t, row.Scan(&count))
		require.Equal(t, 1, count)

		spans := recorder.Ended()
		require.Equal(t, "GetCount", spans[len(spans)-1].Name())
	})

	t.Run("should name a batch query span after sqlc when batch has one statement", func(t *testing.T) {
		batch := &pgx.Batch{}
		batch.Queue("-- name: ListAuthors :many\nSELECT 1")
		require.NoError(t, pool.SendBatch(namedCtx, batch).Close())

		var names []string
		for _, span := range recorder.Ended() {
			names = append(names, span.Name())
		}
		require.Contains(t, names, "ListAuthors")
	})

	t.Run("should name query spans after sqlc when batch has multiple statements", func(t *testing.T) {
		batch := &pgx.Batch{}
		batch.Queue("-- name: LoadFirst :one\nSELECT 1")
		batch.Queue("-- name: LoadSecond :one\nSELECT 2")
		results := pool.SendBatch(namedCtx, batch)

		var result int
		require.NoError(t, results.QueryRow().Scan(&result))
		require.NoError(t, results.QueryRow().Scan(&result))
		require.NoError(t, results.Close())

		var names []string
		for _, span := range recorder.Ended() {
			names = append(names, span.Name())
		}
		require.Contains(t, names, "LoadFirst")
		require.Contains(t, names, "LoadSecond")
	})

	t.Run("should use unmodified SQL as span name when context has no extractor", func(t *testing.T) {
		sql := "-- name: GetAuthor :one\nSELECT 1"
		_, err := pool.Exec(ctx, sql)
		require.NoError(t, err)

		spans := recorder.Ended()
		require.Equal(t, sqlcomment.Append(ctx, sql), spans[len(spans)-1].Name())
	})
}
