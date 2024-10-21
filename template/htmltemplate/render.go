package htmltemplate

import (
	"fmt"
	"io"
	"io/fs"
)

// Renderer renders a template using provided underlying engine.
type Renderer interface {
	// Render renders a template with the given name and data.
	Render(w io.Writer, name string, data any) error
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
	return r.engine.Execute(w, name, data)
}
