package dbsql

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.inout.gg/foundations/must"
)

const (
	serverDurationMetric    = "db.postgresql.server_execution.duration"
	serverDurationAttribute = "db.postgresql.server_execution.duration"
	serverDurationContext   = "go.inout.gg/foundations/dbsql/serverduration/active"
)

type serverDurationCallbackKey struct{}

// WithServerDurationCallback receives each statement's server-measured duration
// on the calling goroutine, before pgx finishes reading the response. A batch
// may call it more than once. The callback must not query on the same connection.
func WithServerDurationCallback(ctx context.Context, callback func(time.Duration)) context.Context {
	return context.WithValue(ctx, serverDurationCallbackKey{}, callback)
}

// WithServerDuration records auto_explain NOTICE durations in OTel traces and a
// histogram (in seconds). It does not load auto_explain or set server parameters;
// the server must already emit NOTICE-level text plans for these connections.
func WithServerDuration() func(*pgxpool.Config) {
	return func(cfg *pgxpool.Config) {
		histogram := must.Must(otel.Meter(meterName).Float64Histogram(
			serverDurationMetric,
			metric.WithUnit("s"),
			metric.WithDescription("PostgreSQL server-measured statement execution duration"),
		))
		observer := &durationObserver{histogram: histogram}

		previousNotice := cfg.ConnConfig.OnNotice
		cfg.ConnConfig.OnNotice = func(conn *pgconn.PgConn, notice *pgconn.Notice) {
			observer.onNotice(conn, notice)
			if previousNotice != nil {
				previousNotice(conn, notice)
			}
		}

		if cfg.ConnConfig.Tracer == nil {
			cfg.ConnConfig.Tracer = observer
		} else {
			cfg.ConnConfig.Tracer = multitracer.New(cfg.ConnConfig.Tracer, observer)
		}
	}
}

type durationObserver struct {
	histogram metric.Float64Histogram
}

func (o *durationObserver) TraceQueryStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	ctx, _ = otel.Tracer(meterName).Start(ctx, "postgresql.query")
	conn.PgConn().CustomData()[serverDurationContext] = ctx
	return ctx
}

func (o *durationObserver) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	delete(conn.PgConn().CustomData(), serverDurationContext)
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
	}
	span.End()
}

func (o *durationObserver) TraceBatchStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	ctx, _ = otel.Tracer(meterName).Start(ctx, "postgresql.batch")
	conn.PgConn().CustomData()[serverDurationContext] = ctx
	return ctx
}

func (o *durationObserver) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

func (o *durationObserver) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	delete(conn.PgConn().CustomData(), serverDurationContext)
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
	}
	span.End()
}

func (o *durationObserver) onNotice(conn *pgconn.PgConn, notice *pgconn.Notice) {
	seconds, ok := parseServerDuration(notice)
	if !ok {
		return
	}

	value, ok := conn.CustomData()[serverDurationContext]
	if !ok {
		return
	}
	ctx := value.(context.Context)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Float64(serverDurationAttribute, seconds))
	span.AddEvent("postgresql.server_execution", trace.WithAttributes(
		attribute.Float64(serverDurationAttribute, seconds),
	))
	o.histogram.Record(ctx, seconds)
	if callback, ok := ctx.Value(serverDurationCallbackKey{}).(func(time.Duration)); ok && callback != nil {
		callback(time.Duration(math.Round(seconds * float64(time.Second))))
	}
}

func parseServerDuration(notice *pgconn.Notice) (float64, bool) {
	if notice == nil || notice.Severity != "NOTICE" {
		return 0, false
	}

	const prefix = "duration: "
	const suffix = " ms  plan:\nQuery Text: "
	if !strings.HasPrefix(notice.Message, prefix) {
		return 0, false
	}
	value, rest, ok := strings.Cut(strings.TrimPrefix(notice.Message, prefix), suffix)
	if !ok || rest == "" {
		return 0, false
	}
	milliseconds, err := strconv.ParseFloat(value, 64)
	if err != nil || milliseconds < 0 || math.IsNaN(milliseconds) || math.IsInf(milliseconds, 0) {
		return 0, false
	}
	return milliseconds / 1000, true
}
