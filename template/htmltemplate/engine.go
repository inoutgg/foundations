package htmltemplate

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

var _ Engine = (*engine)(nil)

// Engine is an adapter for a templating engine that is used by a Renderer.
type Engine interface {
	// ParseFS loads templates from the given fs.
	ParseFS(f fs.FS) error

	// Execute evaluates template with given name and the data context.
	Execute(w io.Writer, name string, data any) error
}

type engine struct {
	root      string
	extension string
	template  *template.Template
}

type EngineConfig struct {
	Root      string
	Extension string
	Funcs     template.FuncMap
}

// NewEngine creates a a new Engine backed by template/html.
func NewEngine(config *EngineConfig) Engine {
	tpl := template.New(config.Root).Funcs(config.Funcs)

	return &engine{
		root:      filepath.ToSlash(config.Root),
		extension: config.Extension,
		template:  tpl,
	}
}

func (e *engine) ParseFS(f fs.FS) error {
	return fs.WalkDir(f, e.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("htmltemplate: failed to parse templates directory: %w", err)
		}

		if entry.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(e.root, path)
		if err != nil {
			return fmt.Errorf("htmltemplate: failed to parse templates from directory: %w", err)
		}

		name := strings.TrimSuffix(filepath.ToSlash(rel), e.extension)
		tpl := e.template.New(name)
		content, err := fs.ReadFile(f, path)
		if err != nil {
			return fmt.Errorf("htmltemplate: failed to read template %q: %w", name, err)
		}

		if _, err := tpl.Parse(string(content)); err != nil {
			return fmt.Errorf("htmltemplate: failed to parse template %q: %w", name, err)
		}

		return nil
	})
}

func (e *engine) Execute(w io.Writer, name string, vars interface{}) error {
	tpl, err := e.lookup(name)
	if err != nil {
		return fmt.Errorf("htmltemplate: failed to execute template %q: %w", name, err)
	}

	err = tpl.Execute(w, vars)
	if err != nil {
		return fmt.Errorf("htmltemplate: failed to execute template %q: %w", name, err)
	}

	return nil
}

// lookup resolves a template from the template tree with given name.
func (e *engine) lookup(name string) (*template.Template, error) {
	tpl := e.template.Lookup(name)
	if tpl == nil {
		return nil, fmt.Errorf("htmltemplate: failed to resolve template %q", name)
	}

	return tpl, nil
}
