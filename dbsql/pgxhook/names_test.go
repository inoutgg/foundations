package pgxhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryNameExtractor_SQLC(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		sql  string
		want string
	}{
		{
			"should return the query name when sqlc annotation leads the SQL",
			"-- name: GetAuthor :one\nSELECT 1", "GetAuthor",
		},
		{
			"should accept leading whitespace when sqlc annotation starts the query",
			"  \n-- name: ListAuthors :many\nSELECT 1", "ListAuthors",
		},
		{"should preserve SQL when query has no sqlc annotation", "SELECT 1", "SELECT 1"},
		{"should preserve SQL when sqlc annotation is malformed", "-- name: GetAuthor\nSELECT 1", "-- name: GetAuthor\nSELECT 1"},
		{
			"should preserve SQL when sqlc annotation follows another statement",
			"SELECT 1\n-- name: GetAuthor :one", "SELECT 1\n-- name: GetAuthor :one",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.want, SQLCQueryName(context.Background(), test.sql))
		})
	}
}

func TestQueryNameExtractor_SpanNameFromContext(t *testing.T) {
	t.Parallel()

	t.Run("should preserve SQL when context has no bound extractor", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "SELECT 1", SpanNameFromContext(context.Background(), "SELECT 1"))
	})

	t.Run("should use sqlc name when context has a bound extractor", func(t *testing.T) {
		t.Parallel()

		ctx := WithQueryNameExtractor(context.Background(), SQLCQueryName)
		require.Equal(t, "GetAuthor", SpanNameFromContext(ctx, "-- name: GetAuthor :one\nSELECT 1"))
	})
}
