package shutdown

import (
	"context"

	"golang.org/x/sync/errgroup"
)

var _ Starter = (*Registry)(nil)

type Registry struct {
	starters []Starter
}

func (r *Registry) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, s := range r.starters {
		g.Go(func() error { return s.Start(ctx) })
	}

	return g.Wait()
}

func (r *Registry) Shutdown(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, s := range r.starters {
		g.Go(func() error { return s.Shutdown(ctx) })
	}

	return g.Wait()
}
