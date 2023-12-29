package otel

import (
	"context"

	"github.com/jackc/pgx/v5"
)

var _ QueryTracer = (*queryTracer)(nil)

type queryTracer struct{}

func (qt *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.TODO()
}

func (qt *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	return
}
