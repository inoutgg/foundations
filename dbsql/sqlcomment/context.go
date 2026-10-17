package sqlcomment

import (
	"context"
	"maps"
)

type (
	ctxOptions struct {
		tags Tags
		skip bool
	}
	ctxKeyType struct{}
)

var ctxOptionsKey ctxKeyType //nolint:gochecknoglobals

// Skip returns a new context that tells the otelc hook to skip SQL commenting.
func Skip(ctx context.Context) context.Context {
	c, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	if !ok {
		return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{skip: true, tags: nil})
	}

	return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{skip: true, tags: c.tags})
}

// WithTags returns a new context with tags merged over any existing tags.
// The supplied tags are copied and override values for existing keys.
func WithTags(ctx context.Context, tags Tags) context.Context {
	previous, _ := ctx.Value(ctxOptionsKey).(*ctxOptions)
	size := len(tags)
	skip := false

	if previous != nil {
		size += len(previous.tags)
		skip = previous.skip
	}

	merged := make(Tags, size)
	if previous != nil {
		maps.Copy(merged, previous.tags)
	}

	maps.Copy(merged, tags)

	return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{tags: merged, skip: skip})
}

// WithTag stores a single key and value on a new context.
func WithTag(ctx context.Context, key, val string) context.Context {
	return WithTags(ctx, Tags{key: val})
}

// FromContext returns a copy of the SQL comment tags stored in ctx, if any.
func FromContext(ctx context.Context) Tags {
	t, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	if !ok {
		return Tags{}
	}

	return maps.Clone(t.tags)
}

func skipFromContext(ctx context.Context) bool {
	t, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	return ok && t.skip
}
