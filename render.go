package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

// Renderer compiles layouts, pages and partials once, at registration time.
// Register everything before serving; the compiled sets are read-only after
// that.
type Renderer struct {
	fsys   fs.FS
	funcs  template.FuncMap
	shared *template.Template
	names  map[string]struct{}
	pages  map[string]*template.Template
	frozen bool
}

// New creates a renderer over fsys. The provided funcs are merged over the
// default helpers; app entries win. fsys must be non-nil.
func New(fsys fs.FS, funcs template.FuncMap) *Renderer {
	merged := defaultFuncs()
	for name, fn := range funcs {
		merged[name] = fn
	}
	return &Renderer{
		fsys:   fsys,
		funcs:  merged,
		shared: template.New("__render_shared").Funcs(merged),
		names:  make(map[string]struct{}),
		pages:  make(map[string]*template.Template),
	}
}

// Partial registers a plain template fragment under name. Partials are
// available to RenderPartial, to page templates and to other partials. They
// must be registered before pages.
func (r *Renderer) Partial(name, path string) error {
	if r.fsys == nil {
		return fmt.Errorf("render: nil fs.FS")
	}
	if r.frozen {
		return fmt.Errorf("render: partial %q registered after a page; register partials before pages", name)
	}
	if err := validName("partial", name); err != nil {
		return err
	}
	src, err := fs.ReadFile(r.fsys, path)
	if err != nil {
		return fmt.Errorf("render: read partial %q (%s): %w", name, path, err)
	}
	tmp, err := template.New(name).Funcs(r.funcs).Parse(string(src))
	if err != nil {
		return fmt.Errorf("render: parse partial %q (%s): %w", name, path, err)
	}
	for _, tmpl := range tmp.Templates() {
		if err := validName("partial", tmpl.Name()); err != nil {
			return fmt.Errorf("render: partial %q (%s): %w", name, path, err)
		}
		if _, exists := r.names[tmpl.Name()]; exists {
			return fmt.Errorf("render: partial %q (%s): duplicate template name %q", name, path, tmpl.Name())
		}
	}
	for _, tmpl := range tmp.Templates() {
		if _, err := r.shared.AddParseTree(tmpl.Name(), tmpl.Tree); err != nil {
			return fmt.Errorf("render: add partial %q (%s): %w", name, path, err)
		}
		r.names[tmpl.Name()] = struct{}{}
	}
	return nil
}

func validName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("render: empty %s name", kind)
	}
	if name == "content" {
		return fmt.Errorf("render: %s name %q is reserved", kind, name)
	}
	if strings.HasPrefix(name, "__render_") {
		return fmt.Errorf("render: %s name %q uses the reserved __render_ prefix", kind, name)
	}
	return nil
}
