package dbsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.inout.gg/foundations/metrics"
	"go.inout.gg/foundations/must"
)

const meterName = "go.inout.gg/foundations/dbsql"

type poolStats struct {
	meter                   metric.Meter
	acquireCount            metric.Int64ObservableCounter
	acquiredConns           metric.Int64ObservableCounter
	canceledAcquireCount    metric.Int64ObservableCounter
	constructingConns       metric.Int64ObservableCounter
	emptyAcquireCount       metric.Int64ObservableCounter
	idleConns               metric.Int64ObservableCounter
	maxConns                metric.Int64ObservableCounter
	maxIdleDestroyCount     metric.Int64ObservableCounter
	maxLifetimeDestroyCount metric.Int64ObservableCounter
	newConnsCount           metric.Int64ObservableCounter
	totalConns              metric.Int64ObservableCounter
}

func newPoolStats(meter metric.Meter) *poolStats {
	return &poolStats{
		meter: meter,
		acquireCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("acquire_count"),
			metric.WithDescription("Number of acquire operations"))),
		acquiredConns: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("acquired_conns"),
			metric.WithDescription("Number of acquired connections"))),
		canceledAcquireCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("canceled_acquire_count"),
			metric.WithDescription("Number of canceled acquire operations"))),
		constructingConns: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("constructing_conns"),
			metric.WithDescription("Number of connections being constructed"))),
		emptyAcquireCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("empty_acquire_count"),
			metric.WithDescription("Number of empty acquire operations"))),
		idleConns: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("idle_conns"),
			metric.WithDescription("Number of idle connections"))),
		maxConns: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("max_conns"),
			metric.WithDescription("Maximum number of connections"))),
		maxIdleDestroyCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("max_idle_destroy_count"),
			metric.WithDescription("Number of connections destroyed due to max idle"))),
		maxLifetimeDestroyCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("max_lifetime_destroy_count"),
			metric.WithDescription("Number of connections destroyed due to max lifetime"))),
		newConnsCount: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("new_conns_count"),
			metric.WithDescription("Number of new connections"))),
		totalConns: must.Must(meter.Int64ObservableCounter(
			metrics.FormatMetricName("total_conns_count"),
			metric.WithDescription("Total number of connections"))),
	}
}

func registerPoolMetrics(pool *pgxpool.Pool) error {
	return newPoolStats(otel.GetMeterProvider().Meter(meterName)).register(pool)
}

func (s *poolStats) register(pool *pgxpool.Pool) error {
	_, err := s.meter.RegisterCallback(
		func(_ context.Context, observer metric.Observer) error {
			stats := pool.Stat()

			observer.ObserveInt64(s.acquireCount, stats.AcquireCount())
			observer.ObserveInt64(s.acquiredConns, int64(stats.AcquiredConns()))
			observer.ObserveInt64(s.canceledAcquireCount, stats.CanceledAcquireCount())
			observer.ObserveInt64(s.constructingConns, int64(stats.ConstructingConns()))
			observer.ObserveInt64(s.emptyAcquireCount, stats.EmptyAcquireCount())
			observer.ObserveInt64(s.idleConns, int64(stats.IdleConns()))
			observer.ObserveInt64(s.maxConns, int64(stats.MaxConns()))
			observer.ObserveInt64(s.maxIdleDestroyCount, stats.MaxIdleDestroyCount())
			observer.ObserveInt64(s.maxLifetimeDestroyCount, stats.MaxLifetimeDestroyCount())
			observer.ObserveInt64(s.newConnsCount, stats.NewConnsCount())
			observer.ObserveInt64(s.totalConns, int64(stats.TotalConns()))

			return nil
		},
		s.acquireCount,
		s.acquiredConns,
		s.canceledAcquireCount,
		s.constructingConns,
		s.emptyAcquireCount,
		s.idleConns,
		s.maxConns,
		s.maxIdleDestroyCount,
		s.maxLifetimeDestroyCount,
		s.newConnsCount,
		s.totalConns,
	)
	if err != nil {
		return fmt.Errorf("dbsql: failed to register pool metrics: %w", err)
	}

	return nil
}
