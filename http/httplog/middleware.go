package httplog

import (
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"

	"go.inout.gg/foundations/http/httpmiddleware"
)

// Middleware logs incoming requests with the provided logger l.
func Middleware(l *slog.Logger) httpmiddleware.MiddlewareFunc {
	return httpmiddleware.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			metrics := httpsnoop.CaptureMetrics(next, w, r)
			l.InfoContext(
				ctx,
				"incoming request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", metrics.Code),
				slog.Duration("duration", metrics.Duration),
				slog.Int64("bytes", metrics.Written),
			)
		})
	})
}
