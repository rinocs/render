package render

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

// RenderPartial executes the registered partial name. On an execution error
// nothing is written to w; on success it sets the HTML content type and
// writes the rendered bytes.
func (r *Renderer) RenderPartial(w http.ResponseWriter, name string, data any) error {
	tmpl := r.shared.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: partial %q not found", name)
	}
	return execute(w, "partial "+name, tmpl, data)
}

func execute(w http.ResponseWriter, label string, tmpl *template.Template, data any) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render: execute %s: %w", label, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("render: write %s: %w", label, err)
	}
	return nil
}

// RenderPage executes the registered page name. On an execution error nothing
// is written to w; on success it sets the HTML content type and writes the
// rendered bytes.
func (r *Renderer) RenderPage(w http.ResponseWriter, name string, data any) error {
	tmpl := r.pages[name]
	if tmpl == nil {
		return fmt.Errorf("render: page %q not found", name)
	}
	return execute(w, "page "+name, tmpl, data)
}
