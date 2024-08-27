package html

import (
	"context"
	"errors"
	"net/http"

	"go.inout.gg/foundations/http/middleware"
	"go.inout.gg/foundations/must"
)

type ctxKey struct{}

var kCtxKey = ctxKey{}

func Middleware(r Renderer) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), kCtxKey, r)))
		})
	}
}

// Render renders an HTML template with the given name and variables.
func Render(w http.ResponseWriter, req *http.Request, name string, vars ...any) error {
	render, ok := req.Context().Value(kCtxKey).(Renderer)
	if !ok {
		return errors.New(
			"http/render: unable to retrieve render from request context. Make sure to use corresponding middleware.",
		)
	}

	if err := render.Render(w, name, vars); err != nil {
		return err
	}

	return nil
}

// MustRender is like Render, but panics if an error occurs.
func MustRender(w http.ResponseWriter, req *http.Request, name string, vars ...any) {
	must.Must1(Render(w, req, name, vars...))
}
