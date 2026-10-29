package render

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestPartialMissingFile(t *testing.T) {
	r := New(fstest.MapFS{}, nil)
	err := r.Partial("row", "partials/missing.html")
	if err == nil {
		t.Fatal("expected an error for a missing partial file")
	}
	if !strings.Contains(err.Error(), "partials/missing.html") {
		t.Fatalf("error does not name the missing file: %v", err)
	}
}

func TestPartialDuplicate(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/row.html":   &fstest.MapFile{Data: []byte(`<li>{{.}}</li>`)},
		"partials/other.html": &fstest.MapFile{Data: []byte(`<p>other</p>`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("row", "partials/row.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Partial("row", "partials/other.html"); err == nil {
		t.Fatal("expected an error for a duplicate partial name")
	}
}

func TestPartialReservedContent(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/row.html": &fstest.MapFile{Data: []byte(`<li>{{.}}</li>`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("content", "partials/row.html"); err == nil {
		t.Fatal("expected an error for the reserved partial name content")
	}
}

func TestPartialIncludesPartial(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/item.html": &fstest.MapFile{Data: []byte(`<li>{{.}}</li>`)},
		"partials/list.html": &fstest.MapFile{Data: []byte(`<ul>{{range .}}{{template "item" .}}{{end}}</ul>`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("item", "partials/item.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Partial("list", "partials/list.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "list", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "<ul><li>a</li><li>b</li></ul>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestNilFS(t *testing.T) {
	r := New(nil, nil)
	if err := r.Partial("row", "partials/row.html"); err == nil {
		t.Fatal("expected an error for a nil fs.FS")
	}
}
