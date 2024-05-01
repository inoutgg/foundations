package dbotel

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

var _ pgx.QueryTracer = (*tracer)(nil)
var _ pgx.BatchTracer = (*tracer)(nil)
var _ pgx.ConnectTracer = (*tracer)(nil)
var _ pgx.PrepareTracer = (*tracer)(nil)
var _ pgx.CopyFromTracer = (*tracer)(nil)

type tracer struct {
	tracer trace.Tracer
}

// NewTracer returns a new tracer implementing OTEL instrumentation for the
// pgx driver.
func NewTracer() pgx.QueryTracer {
	return &tracer{}
}

// TraceBatchEnd implements pgx.BatchTracer.
func (qt *tracer) TraceBatchEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchEndData,
) {
	panic("unimplemented")
}

// TraceBatchQuery implements pgx.BatchTracer.
func (qt *tracer) TraceBatchQuery(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchQueryData,
) {
	panic("unimplemented")
}

// TraceBatchStart implements pgx.BatchTracer.
func (qt *tracer) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	panic("unimplemented")
}

// TraceConnectEnd implements pgx.ConnectTracer.
func (qt *tracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	panic("unimplemented")
}

// TraceConnectStart implements pgx.ConnectTracer.
func (qt *tracer) TraceConnectStart(
	ctx context.Context,
	data pgx.TraceConnectStartData,
) context.Context {
	panic("unimplemented")
}

// TracePrepareEnd implements pgx.PrepareTracer.
func (qt *tracer) TracePrepareEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareEndData,
) {
	panic("unimplemented")
}

// TracePrepareStart implements pgx.PrepareTracer.
func (qt *tracer) TracePrepareStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	panic("unimplemented")
}

// TraceCopyFromEnd implements pgx.CopyFromTracer.
func (qt *tracer) TraceCopyFromEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromEndData,
) {
	panic("unimplemented")
}

// TraceCopyFromStart implements pgx.CopyFromTracer.
func (qt *tracer) TraceCopyFromStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	panic("unimplemented")
}

func (qt *tracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	return context.TODO()
}

func (qt *tracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	return
}
