package htmlrender

import (
	"net/http"

	"github.com/atcirclesquare/common/template/html"
)

type render struct {
	engine html.Renderer
}

func (r *render) Render(w http.ResponseWriter, name string, data any) error {
	return r.engine.Render(w, name, data)
}
