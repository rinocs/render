package render

import (
	"html/template"
	"testing"
	"testing/fstest"
)

func TestDefaultFuncs(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/funcs.html": &fstest.MapFile{Data: []byte(`{{join ", " .Tags}}|{{safeHTML .HTML}}|{{$m := dict "k" "v"}}{{$m.k}}`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("funcs", "partials/funcs.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "funcs", map[string]any{
		"Tags": []string{"a", "b"},
		"HTML": "<b>x</b>",
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), `a, b|<b>x</b>|v`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestFuncsOverrideDefaults(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/join.html": &fstest.MapFile{Data: []byte(`{{join ", " .Tags}}`)},
	}
	r := New(fsys, template.FuncMap{
		"join": func(string, []string) string { return "override" },
	})
	if err := r.Partial("join", "partials/join.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "join", map[string]any{"Tags": []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "override"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
