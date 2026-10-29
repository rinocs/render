package render

import (
	"errors"
	"fmt"
	"html/template"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestRenderPageWithLayout(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "home", map[string]any{"Title": "Home", "Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	for _, want := range []string{"<title>Home</title>", "<p>Ada</p>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderStandalonePage(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("raw", "standalone.html", ""); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "raw", map[string]any{"Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "<p>Ada</p>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestPageIncludesPartial(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Partial("row", "partials/row.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Page("list", "page-with-row.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "list", map[string]any{"Title": "List", "Rows": []string{"a", "b"}}); err != nil {
		t.Fatal(err)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<ul><li>a</li><li>b</li></ul>") {
		t.Fatalf("body missing rendered rows: %s", body)
	}
}

func TestPageUsesLayoutDefine(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("nav", "page-with-nav.html", "layout-with-nav.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "nav", nil); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "<main><section><nav>menu</nav></section></main>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestLayoutIncludesPartial(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Partial("banner", "partials/banner.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Page("home", "page.html", "layout-with-banner.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "home", map[string]any{"Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	for _, want := range []string{"<b>Promo</b>", "<p>Ada</p>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderPageMissing(t *testing.T) {
	r := New(pageFS(), nil)
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "missing", nil); err == nil {
		t.Fatal("expected an error for a missing page")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestRenderPageExecutionError(t *testing.T) {
	r := New(pageFS(), template.FuncMap{
		"boom": func() (string, error) { return "", errors.New("boom") },
	})
	if err := r.Page("boom", "page-boom.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "boom", nil); err == nil {
		t.Fatal("expected an execution error")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty", got)
	}
}

func TestEscapingAndSafeHTML(t *testing.T) {
	fsys := fstest.MapFS{
		"partials/esc.html":  &fstest.MapFile{Data: []byte(`<p>{{.HTML}}</p>`)},
		"partials/safe.html": &fstest.MapFile{Data: []byte(`<p>{{safeHTML .HTML}}</p>`)},
	}
	r := New(fsys, nil)
	if err := r.Partial("esc", "partials/esc.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Partial("safe", "partials/safe.html"); err != nil {
		t.Fatal(err)
	}
	data := map[string]any{"HTML": "<b>x</b>"}
	rec := newRecorder(t)
	if err := r.RenderPartial(rec, "esc", data); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "<p>&lt;b&gt;x&lt;/b&gt;</p>"; got != want {
		t.Fatalf("escaped body = %q, want %q", got, want)
	}
	rec = newRecorder(t)
	if err := r.RenderPartial(rec, "safe", data); err != nil {
		t.Fatal(err)
	}
	if got, want := rec.Body.String(), "<p><b>x</b></p>"; got != want {
		t.Fatalf("safeHTML body = %q, want %q", got, want)
	}
}

func TestConcurrentRender(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("user-%d", i)
			rec := httptest.NewRecorder()
			if err := r.RenderPage(rec, "home", map[string]any{"Title": "Home", "Name": name}); err != nil {
				errs <- err
				return
			}
			if !strings.Contains(rec.Body.String(), name) {
				errs <- fmt.Errorf("missing %s", name)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestLayoutIfWithoutElse(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout-with-if.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "home", map[string]any{"Title": "Home", "Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<p>Ada</p>") {
		t.Fatalf("body = %s", body)
	}
}
