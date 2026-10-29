package render

import (
	"errors"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newRecorder(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	return httptest.NewRecorder()
}

func TestRenderPartial(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/row.html": &fstest.MapFile{Data: []byte(`<li>{{.Name}}</li>`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("row", "partials/row.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "row", map[string]any{"Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Body.String(); got != "<li>Ada</li>" {
		t.Fatalf("body = %q", got)
	}
}

func TestRenderPartialMissing(t *testing.T) {
	r := New(fstest.MapFS{}, nil)
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "missing", nil); err == nil {
		t.Fatal("expected an error for a missing partial")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty", got)
	}
}

func TestRenderPartialExecutionError(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/boom.html": &fstest.MapFile{Data: []byte(`{{boom}}`)},
	}
	r := New(fsys, template.FuncMap{
		"boom": func() (string, error) { return "", errors.New("boom") },
	})
	if err := r.Partial("boom", "partials/boom.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	err := r.RenderPartial(rec, "boom", nil)
	if err == nil {
		t.Fatal("expected an execution error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error does not name the partial: %v", err)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty", got)
	}
}
