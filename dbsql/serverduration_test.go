package dbsql

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.inout.gg/foundations/test/testutil"
)

func TestParseServerDuration(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		notice  *pgconn.Notice
		seconds float64
		valid   bool
	}{
		{"auto_explain", &pgconn.Notice{Severity: "NOTICE", Message: "duration: 12.345 ms  plan:\nQuery Text: select 1\nResult"}, 0.012345, true},
		{"zero", &pgconn.Notice{Severity: "NOTICE", Message: "duration: 0.000 ms  plan:\nQuery Text: select 1\nResult"}, 0, true},
		{"other_notice", &pgconn.Notice{Severity: "NOTICE", Message: "something else"}, 0, false},
		{"server_log", &pgconn.Notice{Severity: "LOG", Message: "duration: 12.345 ms  plan:\nQuery Text: select 1"}, 0, false},
		{"invalid_duration", &pgconn.Notice{Severity: "NOTICE", Message: "duration: invalid ms  plan:\nQuery Text: select 1"}, 0, false},
		{"negative_duration", &pgconn.Notice{Severity: "NOTICE", Message: "duration: -1 ms  plan:\nQuery Text: select 1"}, 0, false},
		{"no_query", &pgconn.Notice{Severity: "NOTICE", Message: "duration: 1 ms  plan:\nQuery Text: "}, 0, false},
		{"nil", nil, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			seconds, valid := parseServerDuration(test.notice)
			require.Equal(t, test.valid, valid)
			require.InDelta(t, test.seconds, seconds, 0.000000001)
		})
	}
}

type durationQueryRecorder struct{ started int }

// Test connections configure auto_explain independently of the receive-only
// pool option. Production sessions must be configured by the database admin.
func configureTestAutoExplain(cfg *pgxpool.Config) {
	cfg.ConnConfig.RuntimeParams["session_preload_libraries"] = "auto_explain"
	cfg.ConnConfig.RuntimeParams["auto_explain.log_min_duration"] = "0"
	cfg.ConnConfig.RuntimeParams["auto_explain.log_level"] = "notice"
	cfg.ConnConfig.RuntimeParams["auto_explain.log_format"] = "text"
	cfg.ConnConfig.RuntimeParams["auto_explain.log_analyze"] = "off"
	cfg.ConnConfig.RuntimeParams["client_min_messages"] = "notice"
}

func TestWithServerDurationDoesNotSetServerParameters(t *testing.T) {
	t.Parallel()
	cfg, err := pgxpool.ParseConfig("postgres://test@localhost/test")
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams = map[string]string{"application_name": "api"}

	WithServerDuration()(cfg)

	require.Equal(t, map[string]string{"application_name": "api"}, cfg.ConnConfig.RuntimeParams)
	require.NotNil(t, cfg.ConnConfig.OnNotice)
	require.NotNil(t, cfg.ConnConfig.Tracer)
}

func (r *durationQueryRecorder) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	r.started++
	return ctx
}

func (r *durationQueryRecorder) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestServerDurationTelemetry(t *testing.T) {
	connString := testutil.PgxConnString(t)

	oldMeter := otel.GetMeterProvider()
	oldTracer := otel.GetTracerProvider()
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(oldMeter)
		otel.SetTracerProvider(oldTracer)
		require.NoError(t, meterProvider.Shutdown(context.Background()))
		require.NoError(t, tracerProvider.Shutdown(context.Background()))
	})

	recorder := &durationQueryRecorder{}
	var notices int
	pool, err := NewPool(t.Context(), connString,
		configureTestAutoExplain,
		func(cfg *pgxpool.Config) {
			cfg.ConnConfig.Tracer = recorder
			cfg.ConnConfig.OnNotice = func(_ *pgconn.PgConn, _ *pgconn.Notice) { notices++ }
		},
		WithServerDuration(),
	)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var durations []time.Duration
	ctx := WithServerDurationCallback(t.Context(), func(duration time.Duration) {
		durations = append(durations, duration)
	})
	var result int
	require.NoError(t, pool.QueryRow(ctx, "select 42 from pg_sleep(0.015)").Scan(&result))
	require.Equal(t, 42, result)
	_, err = pool.Exec(ctx, "select pg_sleep(0.01)")
	require.NoError(t, err)

	batch := &pgx.Batch{}
	batch.Queue("select 1")
	batch.Queue("select 2")
	batchResults := pool.SendBatch(ctx, batch)
	require.NoError(t, batchResults.QueryRow().Scan(&result))
	require.Equal(t, 1, result)
	require.NoError(t, batchResults.QueryRow().Scan(&result))
	require.Equal(t, 2, result)
	require.NoError(t, batchResults.Close())

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.QueryRow(ctx, "select 3").Scan(&result))
	require.Equal(t, 3, result)
	require.NoError(t, tx.Commit(ctx))

	require.Len(t, durations, 5)
	require.GreaterOrEqual(t, durations[0], 15*time.Millisecond)
	require.GreaterOrEqual(t, durations[1], 10*time.Millisecond)
	require.GreaterOrEqual(t, notices, 5)
	require.GreaterOrEqual(t, recorder.started, 3)

	spans := spanRecorder.Ended()
	require.GreaterOrEqual(t, len(spans), 4)
	var withDuration int
	for _, span := range spans {
		for _, attr := range span.Attributes() {
			if attr.Key == serverDurationAttribute {
				withDuration++
				require.GreaterOrEqual(t, attr.Value.AsFloat64(), float64(0))
			}
		}
	}
	require.GreaterOrEqual(t, withDuration, 4)

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &collected))
	var count uint64
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name == serverDurationMetric {
				require.Equal(t, "s", m.Unit)
				for _, point := range m.Data.(metricdata.Histogram[float64]).DataPoints {
					count += point.Count
					require.Positive(t, point.Sum)
				}
			}
		}
	}
	require.EqualValues(t, 5, count)
}

func TestServerDurationConnectionLocalAssociation(t *testing.T) {
	connString := testutil.PgxConnString(t)

	pool, err := NewPool(t.Context(), connString, configureTestAutoExplain, WithServerDuration())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	first, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	defer first.Release()
	second, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	defer second.Release()
	require.NotSame(t, first.Conn().PgConn(), second.Conn().PgConn())

	firstDurations := make(chan time.Duration, 2)
	secondDurations := make(chan time.Duration, 2)
	firstCtx := WithServerDurationCallback(t.Context(), func(d time.Duration) { firstDurations <- d })
	secondCtx := WithServerDurationCallback(t.Context(), func(d time.Duration) { secondDurations <- d })

	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, item := range []struct {
		conn *pgxpool.Conn
		ctx  context.Context
	}{
		{first, firstCtx},
		{second, secondCtx},
	} {
		wg.Go(func() {
			var result int
			errors <- item.conn.QueryRow(item.ctx, "select 1 from pg_sleep(0.01)").Scan(&result)
		})
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	require.Len(t, firstDurations, 1)
	require.Len(t, secondDurations, 1)
	require.GreaterOrEqual(t, <-firstDurations, 10*time.Millisecond)
	require.GreaterOrEqual(t, <-secondDurations, 10*time.Millisecond)
	_, ok := first.Conn().PgConn().CustomData()[serverDurationContext]
	require.False(t, ok, "active context must be removed after first query")
	_, ok = second.Conn().PgConn().CustomData()[serverDurationContext]
	require.False(t, ok, "active context must be removed after second query")
}

// BenchmarkServerDuration compares ordinary query throughput with the cost of
// sending a plan NOTICE for every statement. Run with PGX_TEST_DATABASE set.
func BenchmarkServerDuration(b *testing.B) {
	connString := testutil.PgxConnString(b)
	for _, test := range []struct {
		name string
		opts []func(*pgxpool.Config)
	}{
		{name: "baseline"},
		{name: "auto_explain_notices", opts: []func(*pgxpool.Config){configureTestAutoExplain, WithServerDuration()}},
	} {
		b.Run(test.name, func(b *testing.B) {
			pool, err := NewPool(b.Context(), connString, test.opts...)
			require.NoError(b, err)
			defer pool.Close()
			b.ResetTimer()
			for b.Loop() {
				var result int
				require.NoError(b, pool.QueryRow(b.Context(), "select 1").Scan(&result))
			}
		})
	}
}
