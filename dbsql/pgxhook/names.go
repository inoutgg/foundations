package pgxhook

import (
	"context"
	"strings"

	"go.inout.gg/foundations/debug"
)

// QueryNameExtractor derives a span name from a SQL query.
type QueryNameExtractor func(context.Context, string) string

type queryNameExtractorKey struct{}

// WithQueryNameExtractor returns a context that names pgx query spans using
// extractor. Pass it to each query or batch operation, including transactions.
func WithQueryNameExtractor(ctx context.Context, extractor QueryNameExtractor) context.Context {
	debug.Assert(extractor != nil, "pgxhook: query name extractor must not be nil")

	return context.WithValue(ctx, queryNameExtractorKey{}, extractor)
}

// DefaultQueryNameExtractor returns the SQL query unchanged.
func DefaultQueryNameExtractor(_ context.Context, sql string) string {
	return sql
}

// SpanNameFromContext adapts the context-bound extractor for an otelpgx
// WithSpanNameCtxFunc option. Without a bound extractor it uses the default.
func SpanNameFromContext(ctx context.Context, sql string) string {
	if _, ok := ctx.Value(queryNameExtractorKey{}).(QueryNameExtractor); !ok {
		ctx = WithQueryNameExtractor(ctx, DefaultQueryNameExtractor)
	}

	//nolint:forcetypeassert // The binding above guarantees an extractor.
	extractor := ctx.Value(queryNameExtractorKey{}).(QueryNameExtractor)

	return extractor(ctx, sql)
}

// SQLCQueryName returns the name from a leading sqlc query annotation,
// such as "-- name: GetAuthor :one". Other SQL is returned unchanged.
func SQLCQueryName(ctx context.Context, sql string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(sql), "\n")

	annotation, ok := strings.CutPrefix(line, "-- name:")
	if !ok {
		return DefaultQueryNameExtractor(ctx, sql)
	}

	fields := strings.Fields(annotation)
	if len(fields) < 2 || !strings.HasPrefix(fields[1], ":") {
		return DefaultQueryNameExtractor(ctx, sql)
	}

	return fields[0]
}
