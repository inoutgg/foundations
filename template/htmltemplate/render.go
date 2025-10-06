package htmltemplate

import (
	"fmt"
	"io"
	"io/fs"
)

// Renderer renders a template using provided underlying engine.
type Renderer interface {
	// Render renders a template with the given name and data.
	Render(io.Writer, string, any) error
}

type renderer struct {
	engine Engine
}

func New(f fs.FS, e Engine) (Renderer, error) {
	if err := e.ParseFS(f); err != nil {
		return nil, fmt.Errorf("htmltemplate: failed to parse templates: %w", err)
	}

	return &renderer{engine: e}, nil
}

func (r *renderer) Render(w io.Writer, name string, data any) error {
	if err := r.engine.Execute(w, name, data); err != nil {
		return fmt.Errorf("htmltemplate: failed to render template %q: %w", name, err)
	}

	return nil
}
