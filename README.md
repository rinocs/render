# render

Small, stdlib-only HTML rendering for Go. It wraps `html/template` with
layouts, cached template sets, partials and HTMX helpers, without adding a
second template engine.

## Install

```bash
go get github.com/rinocs/render
```

## Quick start

```go
fsys := os.DirFS(".")
r := render.New(fsys, nil)

if err := r.Partial("alert", "templates/partials/alert.html"); err != nil {
	log.Fatal(err)
}
if err := r.Page("home", "templates/pages/home.html", "templates/layouts/base.html"); err != nil {
	log.Fatal(err)
}
if err := r.Page("login", "templates/login.html", ""); err != nil {
	log.Fatal(err)
}

// Full page:
r.RenderPage(w, "home", data)
// Fragment:
r.RenderPartial(w, "alert", data)
// Page for a normal request, fragment for an HTMX request:
r.RenderHTMX(w, req, "home", "alert", data)
```

`RenderPage` and `RenderPartial` return an error and write nothing on a
template execution error. They set `Content-Type: text/html; charset=utf-8` on
success. Status codes and headers such as `Content-Language` belong to the
caller.

## Template conventions

- A **layout** is the root template of its file and calls
  `{{template "content" .}}` directly. It must not wrap itself in `{{define}}`.
- A **page with layout** defines `{{define "content"}}...{{end}}`. Its file
  root is not executed.
- A **standalone page** is a normal root template and is registered with an
  empty layout.
- A **partial** is a plain root fragment, registered under a name. It can be
  rendered from Go, included with `{{template "name" .}}`, and can include
  other partials.
- Register partials before pages. `Partial` after `Page` returns an error.
- `content` and names starting with `__render_` are reserved.
- Layouts can define additional templates (for example a `nav`); pages can use
  them. A page cannot override a name that is already defined by a partial or
  layout.
- One layout level: a layout plus its content. Nested layouts are not
  supported.

Templates are compiled once, at registration. There is no hot reload; restart
the process during development. There is no rendered-output cache.

## HTMX helpers

```go
if render.IsHTMX(req) { ... }
render.HXRedirect(w, "/login")          // sets HX-Redirect, writes no status
render.Redirect(w, req, "/login")       // HTMX: 200 + HX-Redirect; otherwise 303
render.Trigger(w, "job-submitted")      // sets HX-Trigger
```

`RenderHTMX` selects the partial when `HX-Request: true` and the page
otherwise. It never strips a layout from `RenderPage`; `hx-boost` keeps
working.

## Helpers

The default FuncMap contains:

- `dict` — build a `map[string]any` from key/value pairs;
- `safeHTML` — mark a string as trusted HTML;
- `join` — `strings.Join`.

Application funcs passed to `New` override defaults. `safeHTML` performs no
sanitization: only use it for content that is trusted or already sanitized.

## `fs.FS`

`New` accepts any `fs.FS`: `os.DirFS(".")` for templates on disk, or an
`embed.FS` for a single binary. Paths use forward slashes.
