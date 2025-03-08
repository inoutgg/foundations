package sqldbmetrics

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/must"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type stats struct {
	meter                   metric.Meter
	acquireCount            metric.Int64ObservableCounter
	acquireDuration         metric.Int64Histogram
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

func newStats(meter metric.Meter) *stats {
	return &stats{
		meter:                   meter,
		acquireCount:            must.Must(meter.Int64ObservableCounter("acquire_count", metric.WithDescription("Number of acquire operations"))),
		acquireDuration:         must.Must(meter.Int64Histogram("acquire_duration", metric.WithDescription("Duration of acquire operations"))),
		acquiredConns:           must.Must(meter.Int64ObservableCounter("acquired_conns", metric.WithDescription("Number of acquired connections"))),
		canceledAcquireCount:    must.Must(meter.Int64ObservableCounter("canceled_acquire_count", metric.WithDescription("Number of canceled acquire operations"))),
		constructingConns:       must.Must(meter.Int64ObservableCounter("constructing_conns", metric.WithDescription("Number of connections being constructed"))),
		emptyAcquireCount:       must.Must(meter.Int64ObservableCounter("empty_acquire_count", metric.WithDescription("Number of empty acquire operations"))),
		idleConns:               must.Must(meter.Int64ObservableCounter("idle_conns", metric.WithDescription("Number of idle connections"))),
		maxConns:                must.Must(meter.Int64ObservableCounter("max_conns", metric.WithDescription("Maximum number of connections"))),
		maxIdleDestroyCount:     must.Must(meter.Int64ObservableCounter("max_idle_destroy_count", metric.WithDescription("Number of connections destroyed due to max idle"))),
		maxLifetimeDestroyCount: must.Must(meter.Int64ObservableCounter("max_lifetime_destroy_count", metric.WithDescription("Number of connections destroyed due to max lifetime"))),
		newConnsCount:           must.Must(meter.Int64ObservableCounter("new_conns_count", metric.WithDescription("Number of new connections"))),
		totalConns:              must.Must(meter.Int64ObservableCounter("total_conns", metric.WithDescription("Total number of connections"))),
	}
}

func (s *stats) register(p *pgxpool.Pool) error {
	_, err := s.meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		stats := p.Stat()

		// observer.ObserveInt64(s.acquireDuration, stats.AcquireDuration())
		observer.ObserveInt64(s.acquiredConns, int64(stats.AcquiredConns()))
		observer.ObserveInt64(s.canceledAcquireCount, stats.CanceledAcquireCount())
		observer.ObserveInt64(s.constructingConns, int64(stats.ConstructingConns()))
		observer.ObserveInt64(s.emptyAcquireCount, stats.EmptyAcquireCount())
		observer.ObserveInt64(s.idleConns, int64(stats.IdleConns()))
		observer.ObserveInt64(s.maxConns, int64(stats.MaxConns()))
		observer.ObserveInt64(s.maxIdleDestroyCount, stats.MaxIdleDestroyCount())
		observer.ObserveInt64(s.maxLifetimeDestroyCount, stats.MaxLifetimeDestroyCount())
		observer.ObserveInt64(s.newConnsCount, stats.AcquireCount())
		observer.ObserveInt64(s.newConnsCount, stats.NewConnsCount())
		observer.ObserveInt64(s.totalConns, int64(stats.TotalConns()))

		return nil
	})

	return err
}

func MustRegister(p *pgxpool.Pool) {
	var (
		provider = otel.GetMeterProvider()
		meter    = provider.Meter("foundations:sqldbmetrics")
	)

	must.Must1(newStats(meter).register(p))
}
