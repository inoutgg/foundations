package html

import (
	"fmt"
	"io"
	"io/fs"

	"go.inout.gg/common/template/html/engine"
)

// Renderer renders a template using provided underlying engine.
type Renderer interface {
	// Render renders a template with the given name and data.
	Render(w io.Writer, name string, data any) error
}

type renderer struct {
	engine engine.Engine
}

func New(f fs.FS, e engine.Engine) (Renderer, error) {
	if err := e.ParseFS(f); err != nil {
		return nil, fmt.Errorf("template/html: failed to parse templates: %w", err)
	}

	return &renderer{engine: e}, nil
}

func (r *renderer) Render(w io.Writer, name string, data any) error {
	return r.engine.Execute(w, name, data)
}
