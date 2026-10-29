package render

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderHTMXSelectsTemplate(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Partial("fragment", "partials/fragment.html"); err != nil {
		t.Fatal(err)
	}
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}

	hx := httptest.NewRequest(http.MethodGet, "/", nil)
	hx.Header.Set("HX-Request", "true")
	rec := newRecorder(t)
	if err := r.RenderHTMX(rec, hx, "home", "fragment", map[string]any{"Title": "Home", "Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<!doctype") {
		t.Fatalf("HTMX response must be a fragment: %s", body)
	}
	if !strings.Contains(body, "fragment Ada") {
		t.Fatalf("HTMX response missing fragment: %s", body)
	}

	plain := httptest.NewRequest(http.MethodGet, "/", nil)
	rec = newRecorder(t)
	if err := r.RenderHTMX(rec, plain, "home", "fragment", map[string]any{"Title": "Home", "Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<!doctype html>") || !strings.Contains(body, "<p>Ada</p>") {
		t.Fatalf("non-HTMX response must be the full page: %s", body)
	}
}

func TestRenderPageIgnoresHTMXHeader(t *testing.T) {
	r := New(pageFS(), nil)
	if err := r.Page("home", "page.html", "layout.html"); err != nil {
		t.Fatal(err)
	}
	rec := newRecorder(t)
	if err := r.RenderPage(rec, "home", map[string]any{"Title": "Home", "Name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<!doctype html>") {
		t.Fatalf("RenderPage must keep the layout: %s", body)
	}
}

func TestIsHTMX(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"", false},
		{"false", false},
		{"TRUE", false},
	} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.value != "" {
			req.Header.Set("HX-Request", tc.value)
		}
		if got := IsHTMX(req); got != tc.want {
			t.Errorf("IsHTMX(%q) = %t, want %t", tc.value, got, tc.want)
		}
	}
}

func TestHXRedirect(t *testing.T) {
	rec := newRecorder(t)
	HXRedirect(rec, "/login")
	if got, want := rec.Header().Get("HX-Redirect"), "/login"; got != want {
		t.Fatalf("HX-Redirect = %q, want %q", got, want)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRedirect(t *testing.T) {
	hx := httptest.NewRequest(http.MethodGet, "/", nil)
	hx.Header.Set("HX-Request", "true")
	rec := newRecorder(t)
	Redirect(rec, hx, "/login")
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Redirect") != "/login" {
		t.Fatalf("HTMX redirect = %d %q", rec.Code, rec.Header().Get("HX-Redirect"))
	}

	plain := httptest.NewRequest(http.MethodGet, "/", nil)
	rec = newRecorder(t)
	Redirect(rec, plain, "/login")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("plain redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestTrigger(t *testing.T) {
	rec := newRecorder(t)
	Trigger(rec)
	if got := rec.Header().Get("HX-Trigger"); got != "" {
		t.Fatalf("HX-Trigger = %q, want empty", got)
	}
	Trigger(rec, "job-submitted")
	if got, want := rec.Header().Get("HX-Trigger"), "job-submitted"; got != want {
		t.Fatalf("HX-Trigger = %q, want %q", got, want)
	}
	rec = newRecorder(t)
	Trigger(rec, "a", "b")
	if got, want := rec.Header().Get("HX-Trigger"), "a, b"; got != want {
		t.Fatalf("HX-Trigger = %q, want %q", got, want)
	}
}

func TestRenderHTMXMissingTemplate(t *testing.T) {
	r := New(pageFS(), nil)
	hx := httptest.NewRequest(http.MethodGet, "/", nil)
	hx.Header.Set("HX-Request", "true")
	rec := newRecorder(t)
	if err := r.RenderHTMX(rec, hx, "home", "missing", nil); err == nil {
		t.Fatal("expected an error for a missing fragment")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}
