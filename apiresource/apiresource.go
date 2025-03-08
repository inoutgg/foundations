// Package apiresource implements an API resource management as described in
// https://brandur.org/two-phase-render
package apiresource

import (
	"context"
	"fmt"
)

type Renderable[TBundle any, TEntity any, TResource any, TParams any] interface {
	Bundle(context.Context, TParams, []TEntity) (TBundle, error)
	Render(context.Context, TParams, TBundle, TEntity) (TResource, error)
}

// Render renders an API resource.
func Render[
	TRenderable Renderable[TBundle, TEntity, TRenderable, TParams],
	TBundle any,
	TEntity any,
	TParams any,
](
	ctx context.Context,
	params TParams,
	entity TEntity,
) (TRenderable, error) {
	var renderable TRenderable

	bundle, err := renderable.Bundle(ctx, params, []TEntity{entity})
	if err != nil {
		return renderable, fmt.Errorf("apiresource: error loading bundle: %w", err)
	}

	resource, err := renderable.Render(ctx, params, bundle, entity)
	if err != nil {
		return renderable, fmt.Errorf("apiresource: error rendering resource: %w", err)
	}

	return resource, nil
}

// RenderMany is similar to Render, but renders many API resources at once.
func RenderMany[
	TBundle any,
	TRenderable Renderable[TBundle, TEntity, TRenderable, TParams],
	TEntity any,
	TParams any,
](
	ctx context.Context,
	params TParams,
	entities []TEntity,
) ([]TRenderable, error) {
	var renderable TRenderable

	bundle, err := renderable.Bundle(ctx, params, entities)
	if err != nil {
		return nil, fmt.Errorf("apiresource: error loading bundle: %w", err)
	}

	resources := make([]TRenderable, len(entities))
	for i := range resources {
		resources[i], err = renderable.Render(ctx, params, bundle, entities[i])
		if err != nil {
			return nil, fmt.Errorf("apiresource: error rendering resource: %w", err)
		}
	}

	return resources, nil
}
