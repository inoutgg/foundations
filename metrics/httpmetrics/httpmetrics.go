package httpmetrics

import (
	"cmp"
	"net/http"

	"github.com/felixge/httpsnoop"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/metrics"
	"go.inout.gg/foundations/must"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	name = "go.inout.gg/metrics/httpmetrics"
)

type stats struct {
	inflightRequests metric.Int64UpDownCounter
	requestsDuration metric.Int64Histogram
	responseBodySize metric.Int64Histogram
}

func newStats(m metric.Meter) *stats {
	return &stats{
		inflightRequests: must.Must(m.Int64UpDownCounter(
			metrics.FormatMetricName("inflight_requests"),
			metric.WithDescription("The number of inflight requests."),
			metric.WithUnit("{request}"),
		)),
		requestsDuration: must.Must(m.Int64Histogram(
			metrics.FormatMetricName("request_duration_ms"),
			metric.WithDescription("The incoming request duration in milliseconds."),
			metric.WithUnit("ms"),
			metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 200, 500, 1_000, 5_000, 10_000, 30_000, 60_000),
		)),
		responseBodySize: must.Must(m.Int64Histogram(
			metrics.FormatMetricName("request_body_size_bytes"),
			metric.WithDescription("The incoming request body size in bytes."),
			metric.WithUnit("bytes"),
			metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 200, 500, 1_000, 5_000, 10_000, 30_000, 60_000),
		)),
	}
}

func (s *stats) RecordInflightRequest(r *http.Request) func() {
	attrs := attribute.NewSet(
		attribute.String("method", r.Method),
		attribute.String("path", r.URL.Path),
	)

	s.inflightRequests.Add(r.Context(), 1, metric.WithAttributeSet(attrs))

	return func() {
		s.inflightRequests.Add(r.Context(), -1, metric.WithAttributeSet(attrs))
	}
}

func (s *stats) RecordHandledRequest(r *http.Request, metrics httpsnoop.Metrics) {
	ctx := r.Context()
	attrs := attribute.NewSet(
		attribute.Int("code", metrics.Code),
		attribute.String("method", r.Method),
		attribute.String("path", r.URL.Path),
	)

	s.requestsDuration.Record(ctx, metrics.Duration.Milliseconds(), metric.WithAttributeSet(attrs))
	s.responseBodySize.Record(ctx, metrics.Written, metric.WithAttributeSet(attrs))
}

type Config struct {
	Provider metric.MeterProvider
}

func (c *Config) defaults() {
	c.Provider = cmp.Or(c.Provider, otel.GetMeterProvider())
}

// Middleware returns a middleware that captures metrics for incoming HTTP requests.
func Middleware(cfg *Config) httpmiddleware.MiddlewareFunc {
	cfg.defaults()
	debug.Assert(cfg.Provider != nil, "provider is nil")

	var (
		meter = cfg.Provider.Meter(name)
		stats = newStats(meter)
	)

	return httpmiddleware.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			finishInflightRequest := stats.RecordInflightRequest(r)
			defer finishInflightRequest()

			metrics := httpsnoop.CaptureMetrics(next, w, r)
			stats.RecordHandledRequest(r, metrics)
		})
	})
}
