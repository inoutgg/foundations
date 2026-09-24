package sqlcomment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestOTELComments(t *testing.T) {
	t.Run("tagger injects trace context", func(t *testing.T) {
		// Arrange
		old := otel.GetTextMapPropagator()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		defer otel.SetTextMapPropagator(old)
		ctx := testSpanContext()
		want := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

		// Act
		tags := NewOTELTagger().Tag(ctx)

		// Assert
		require.Equal(t, want, tags["traceparent"])
	})

	t.Run("DB adds trace context", func(t *testing.T) {
		// Arrange
		old := otel.GetTextMapPropagator()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		defer otel.SetTextMapPropagator(old)
		wrapped, recorder := newTracedDB(t)
		ctx := testSpanContext()
		want := "select 1 /*traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01'*/"

		// Act
		_, err := wrapped.Exec(ctx, "select 1")

		// Assert
		require.NoError(t, err)
		require.Equal(t, want, recorder.Last(t))
	})
}

func testSpanContext() context.Context {
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
			SpanID:     trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
			TraceFlags: trace.FlagsSampled,
		},
	))
}
