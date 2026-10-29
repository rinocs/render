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

func pageFS() fstest.MapFS {
	return fstest.MapFS{
		"layout.html":             &fstest.MapFile{Data: []byte(`<!doctype html><title>{{.Title}}</title><main>{{template "content" .}}</main>`)},
		"layout-no-content.html":  &fstest.MapFile{Data: []byte(`<p>{{.Title}}</p>`)},
		"layout-with-nav.html":    &fstest.MapFile{Data: []byte(`{{define "nav"}}<nav>menu</nav>{{end}}<main>{{template "content" .}}</main>`)},
		"layout-with-banner.html": &fstest.MapFile{Data: []byte(`<header>{{template "banner" .}}</header><main>{{template "content" .}}</main>`)},
		"page.html":               &fstest.MapFile{Data: []byte(`{{define "content"}}<p>{{.Name}}</p>{{end}}`)},
		"standalone.html":         &fstest.MapFile{Data: []byte(`<p>{{.Name}}</p>`)},
		"page-with-row.html":      &fstest.MapFile{Data: []byte(`{{define "content"}}<ul>{{range .Rows}}{{template "row" .}}{{end}}</ul>{{end}}`)},
		"page-with-nav.html":      &fstest.MapFile{Data: []byte(`{{define "content"}}<section>{{template "nav" .}}</section>{{end}}`)},
		"no-content.html":         &fstest.MapFile{Data: []byte(`<p>no content</p>`)},
		"page-bad-html.html":      &fstest.MapFile{Data: []byte(`{{define "content"}}{{if}}{{end}}`)},
		"page-boom.html":          &fstest.MapFile{Data: []byte(`{{define "content"}}{{boom}}{{end}}`)},
		"partials/row.html":       &fstest.MapFile{Data: []byte(`<li>{{.}}</li>`)},
		"partials/banner.html":    &fstest.MapFile{Data: []byte(`<b>Promo</b>`)},
		"partials/fragment.html":  &fstest.MapFile{Data: []byte(`<p>fragment {{.Name}}</p>`)},
	}
}

func TestPageWithoutContentFails(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("bad", "no-content.html", "layout.html"); err == nil {
		t.Fatal("expected an error for a page without content")
	}
}

func TestLayoutWithoutContentFails(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("bad", "page.html", "layout-no-content.html"); err == nil {
		t.Fatal("expected an error for a layout without a content call")
	}
}

func TestPageMissingFileFails(t *testing.T) {
	r := New(pageFS(), nil)
	err := r.Page("bad", "missing.html", "layout.html")
	if err == nil {
		t.Fatal("expected an error for a missing page file")
	}
	if !strings.Contains(err.Error(), "missing.html") {
		t.Fatalf("error does not name the missing file: %v", err)
	}
}

func TestPageNilFS(t *testing.T) {
	r := New(nil, nil)
	if err := r.Page("home", "page.html", "layout.html"); err == nil {
		t.Fatal("expected an error for a nil fs.FS")
	}
}

func TestPageParseError(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("bad", "page-bad-html.html", "layout.html"); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestPageDuplicateFails(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Page("home", "standalone.html", ""); err == nil {
		t.Fatal("expected an error for a duplicate page name")
	}
}

func TestPageReservedNameFails(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("content", "page.html", "layout.html"); err == nil {
		t.Fatal("expected an error for the reserved page name content")
	}
}

func TestPartialAfterPageFails(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Partial("row", "partials/row.html"); err == nil {
		t.Fatal("expected an error for a partial registered after a page")
	}
}
