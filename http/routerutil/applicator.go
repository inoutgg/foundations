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

type ApplicatorChain struct {
	applicators []Applicator
}

// NewApplicatorChain creates a new chain of router applicators.
func NewApplicatorChain(applicators ...Applicator) ApplicatorChain {
	return ApplicatorChain{applicators}
}

// Apply applies applicators to a given router.
func (c ApplicatorChain) Apply(r chi.Router) chi.Router {
	for _, applicator := range c.applicators {
		r = applicator.Apply(r)
	}

	return r
}
