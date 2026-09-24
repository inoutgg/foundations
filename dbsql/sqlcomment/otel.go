package sqlcomment

import (
	"context"
	"maps"
	"slices"

	"go.opentelemetry.io/otel"
)

type (
	// OTELTagger adds OpenTelemetry propagation fields to SQL comments.
	OTELTagger struct{}

	commentCarrier Tags
)

// NewOTELTagger returns a Tagger which derives SQL comment tags from OpenTelemetry.
func NewOTELTagger() OTELTagger {
	return OTELTagger{}
}

func (ot OTELTagger) Tag(ctx context.Context) Tags {
	carrier := make(commentCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	return Tags(carrier)
}

func (c commentCarrier) Get(key string) string { return c[key] }
func (c commentCarrier) Set(key, value string) { c[key] = value }
func (c commentCarrier) Keys() []string        { return slices.Collect(maps.Values(c)) }
