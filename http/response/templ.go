package response

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

// TEMPL writes the given (*.templ) template to the response.
//
// The template is renderer with a request's context.
func TEMPL(w http.ResponseWriter, r *http.Request, t templ.Component, status int) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)

	err := t.Render(r.Context(), w)
	if err != nil {
		return fmt.Errorf("http/response: unable to render template: %w", err)
	}

	return nil
}
