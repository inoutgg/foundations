package httpmetrics

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Middleware returns a middleware that captures metrics for incoming HTTP requests.
func Middleware(p metric.MeterProvider) httpmiddleware.Middleware {
	meter := p.Meter("foundations:httpmetrics")
	requestDurationHisto := must.Must(
		meter.Int64Histogram(
			"request_duration_ms",
			metric.WithDescription("The incoming request duration in milliseconds."),
			metric.WithUnit("ms"),
			metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 200, 500, 1_000, 5_000, 10_000, 30_000, 60_000),
		),
	)
	responseBodySizeHisto := must.Must(
		meter.Int64Histogram(
			"response_body_size_bytes",
			metric.WithDescription("The outgoing response body size in bytes."),
			metric.WithUnit("bytes"),
			metric.WithExplicitBucketBoundaries(1, 10, 100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000),
		),
	)

	return httpmiddleware.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			metrics := httpsnoop.CaptureMetrics(next, w, r)
			defaultAttributes := attribute.NewSet(
				attribute.Int("code", metrics.Code),
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
			)

			requestDurationHisto.Record(ctx, metrics.Duration.Milliseconds(), metric.WithAttributeSet(defaultAttributes))
			responseBodySizeHisto.Record(ctx, metrics.Written, metric.WithAttributeSet(defaultAttributes))
		})
	})
}
