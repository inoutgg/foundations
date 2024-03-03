package middleware

import "net/http"

var _ Middleware = (*Chain)(nil)

type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new chain of middlewares.
func NewChain(middlewares ...Middleware) Chain {
	return Chain{middlewares: middlewares}
}

// Middleware wraps the given handler with all the middlewares in the chain.
func (c Chain) Middleware(h http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i].Middleware(h)
	}

	return h
}

// Extend returns a new chain of middlewares that contains the middlewares of
// the current chain and the given middlewares.
func (c Chain) Extend(middlewares ...Middleware) Chain {
	return NewChain(append(c.middlewares, middlewares...)...)
}
