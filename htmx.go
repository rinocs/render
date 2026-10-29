package render

import (
	"net/http"
	"strings"
)

// IsHTMX reports whether the request was issued by HTMX.
func IsHTMX(req *http.Request) bool {
	return req.Header.Get("HX-Request") == "true"
}

// HXRedirect sets the HTMX redirect header without writing a status. Callers
// that need a specific status (for example 401) write it themselves.
func HXRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
}

// Redirect sends an HTMX client redirect with 200, or a 303 for a normal
// request.
func Redirect(w http.ResponseWriter, req *http.Request, url string) {
	if IsHTMX(req) {
		HXRedirect(w, url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, req, url, http.StatusSeeOther)
}

// Trigger sets HX-Trigger to the comma-separated events.
func Trigger(w http.ResponseWriter, events ...string) {
	if len(events) == 0 {
		return
	}
	w.Header().Set("HX-Trigger", strings.Join(events, ", "))
}
