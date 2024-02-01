package routerutil

import (
	"github.com/go-chi/chi/v5"
)

var _ Applicator = (ApplicatorFunc)(nil)

// Applicator applies a changes to the router and returns it.
type Applicator interface {
	Apply(chi.Router) chi.Router
}

// ApplicatorFunc is a function that implements the Applicator interface.
type ApplicatorFunc func(chi.Router) chi.Router

func (af ApplicatorFunc) Apply(r chi.Router) chi.Router { return af(r) }

type Chain struct {
	applicators []Applicator
}

// NewChain creates a new chain of router applicators.
func NewChain(applicators ...Applicator) Chain {
	return Chain{applicators}
}

// Apply applies applicators to a given router.
func (c Chain) Apply(r chi.Router) chi.Router {
	for _, applicator := range c.applicators {
		r = applicator.Apply(r)
	}

	return r
}
