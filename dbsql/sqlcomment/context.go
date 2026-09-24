package sqlcomment

import (
	"context"
	"maps"
)

type (
	ctxOptions struct {
		skip bool
		tags Tags
	}
	ctxKeyType struct{}
)

var ctxOptionsKey ctxKeyType //nolint:gochecknoglobals

// Skip returns a new context that tells the DB wrapper to skip SQL commenting.
func Skip(ctx context.Context) context.Context {
	c, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	if !ok {
		return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{skip: true})
	}
	c.skip = true
	return ctx
}

// WithTag stores the key and val pair on the context.
func WithTag(ctx context.Context, key, val string) context.Context {
	t, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	if !ok {
		return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{tags: Tags{key: val}})
	}

	newTags := make(Tags, len(t.tags)+1)
	maps.Copy(newTags, t.tags)
	newTags[key] = val

	return context.WithValue(ctx, ctxOptionsKey, &ctxOptions{
		skip: t.skip,
		tags: newTags,
	})
}

// FromContext returns the SQL comment tags stored in ctx, if any.
func FromContext(ctx context.Context) Tags {
	t, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	if !ok {
		return Tags{}
	}

	return t.tags
}

func skipFromContext(ctx context.Context) bool {
	t, ok := ctx.Value(ctxOptionsKey).(*ctxOptions)
	return ok && t.skip
}
