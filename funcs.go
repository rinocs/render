package render

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
)

func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"dict":     dict,
		"safeHTML": safeHTML,
		"join":     join,
	}
}

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("dict: odd number of arguments")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key %d is not a string", i)
		}
		m[key] = values[i+1]
	}
	return m, nil
}

// safeHTML marks s as trusted HTML. Callers must only pass content that has
// already been sanitized or is trusted by construction.
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func join(sep string, values []string) string {
	return strings.Join(values, sep)
}
