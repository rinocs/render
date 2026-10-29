package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"strings"
	"text/template/parse"
)

// Renderer compiles layouts, pages and partials once, at registration time.
// Register everything before serving; the compiled sets are read-only after
// that.
type Renderer struct {
	fsys   fs.FS
	funcs  template.FuncMap
	shared *template.Template
	names  map[string]struct{}
	pages  map[string]*template.Template
	frozen bool
}

// New creates a renderer over fsys. The provided funcs are merged over the
// default helpers; app entries win. fsys must be non-nil.
func New(fsys fs.FS, funcs template.FuncMap) *Renderer {
	merged := defaultFuncs()
	for name, fn := range funcs {
		merged[name] = fn
	}
	return &Renderer{
		fsys:   fsys,
		funcs:  merged,
		shared: template.New("__render_shared").Funcs(merged),
		names:  make(map[string]struct{}),
		pages:  make(map[string]*template.Template),
	}
}

// Partial registers a plain template fragment under name. Partials are
// available to RenderPartial, to page templates and to other partials. They
// must be registered before pages.
func (r *Renderer) Partial(name, path string) error {
	if r.fsys == nil {
		return fmt.Errorf("render: nil fs.FS")
	}
	if r.frozen {
		return fmt.Errorf("render: partial %q registered after a page; register partials before pages", name)
	}
	if err := validName("partial", name); err != nil {
		return err
	}
	src, err := fs.ReadFile(r.fsys, path)
	if err != nil {
		return fmt.Errorf("render: read partial %q (%s): %w", name, path, err)
	}
	tmp, err := template.New(name).Funcs(r.funcs).Parse(string(src))
	if err != nil {
		return fmt.Errorf("render: parse partial %q (%s): %w", name, path, err)
	}
	for _, tmpl := range tmp.Templates() {
		if err := validName("partial", tmpl.Name()); err != nil {
			return fmt.Errorf("render: partial %q (%s): %w", name, path, err)
		}
		if _, exists := r.names[tmpl.Name()]; exists {
			return fmt.Errorf("render: partial %q (%s): duplicate template name %q", name, path, tmpl.Name())
		}
	}
	for _, tmpl := range tmp.Templates() {
		if _, err := r.shared.AddParseTree(tmpl.Name(), tmpl.Tree); err != nil {
			return fmt.Errorf("render: add partial %q (%s): %w", name, path, err)
		}
		r.names[tmpl.Name()] = struct{}{}
	}
	return nil
}

func validName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("render: empty %s name", kind)
	}
	if name == "content" {
		return fmt.Errorf("render: %s name %q is reserved", kind, name)
	}
	if strings.HasPrefix(name, "__render_") {
		return fmt.Errorf("render: %s name %q uses the reserved __render_ prefix", kind, name)
	}
	return nil
}

// Page registers name from path. With a non-empty layout the layout must be a
// root template that calls {{template "content" .}} and the page must define
// "content". An empty layout registers a standalone page. Pages must be
// registered after partials.
func (r *Renderer) Page(name, path, layout string) error {
	if r.fsys == nil {
		return fmt.Errorf("render: nil fs.FS")
	}
	r.frozen = true
	if err := validName("page", name); err != nil {
		return err
	}
	if _, exists := r.pages[name]; exists {
		return fmt.Errorf("render: page %q already registered", name)
	}

	set, err := r.shared.Clone()
	if err != nil {
		return fmt.Errorf("render: clone templates for page %q: %w", name, err)
	}

	if layout == "" {
		pageSet, err := parseTemplateFile("page", path, "__render_page", r.fsys, r.funcs)
		if err != nil {
			return err
		}
		if err := mergeTemplates(set, pageSet, "__render_page", "page", path, true); err != nil {
			return err
		}
		r.pages[name] = set.Lookup("__render_page")
		return nil
	}

	layoutSet, err := parseTemplateFile("layout", layout, "__render_layout", r.fsys, r.funcs)
	if err != nil {
		return err
	}
	if err := mergeTemplates(set, layoutSet, "__render_layout", "layout", layout, false); err != nil {
		return err
	}

	pageSet, err := parseTemplateFile("page", path, "__render_page", r.fsys, r.funcs)
	if err != nil {
		return err
	}
	if err := mergeTemplates(set, pageSet, "__render_page", "page", path, true); err != nil {
		return err
	}
	if set.Lookup("content") == nil {
		return fmt.Errorf("render: page %q (%s): missing content block", name, path)
	}
	layoutTmpl := set.Lookup("__render_layout")
	if !referencesContent(layoutTmpl) {
		return fmt.Errorf("render: page %q: layout %q does not reference content", name, layout)
	}
	r.pages[name] = layoutTmpl
	return nil
}

func parseTemplateFile(kind, path, rootName string, fsys fs.FS, funcs template.FuncMap) (*template.Template, error) {
	src, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("render: read %s (%s): %w", kind, path, err)
	}
	tmpl, err := template.New(rootName).Funcs(funcs).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("render: parse %s (%s): %w", kind, path, err)
	}
	return tmpl, nil
}

func mergeTemplates(dst, src *template.Template, rootName, kind, path string, allowContent bool) error {
	for _, tmpl := range src.Templates() {
		name := tmpl.Name()
		if name != rootName {
			if name == "content" {
				if !allowContent {
					return fmt.Errorf("render: %s %q defines reserved template %q", kind, path, name)
				}
			} else if strings.HasPrefix(name, "__render_") {
				return fmt.Errorf("render: %s %q uses reserved template name %q", kind, path, name)
			}
		}
		if dst.Lookup(name) != nil {
			return fmt.Errorf("render: %s %q: template name %q already registered", kind, path, name)
		}
		if _, err := dst.AddParseTree(name, tmpl.Tree); err != nil {
			return fmt.Errorf("render: add %s %q (%s): %w", kind, name, path, err)
		}
	}
	return nil
}

func referencesContent(t *template.Template) bool {
	if t == nil || t.Tree == nil || t.Tree.Root == nil {
		return false
	}
	found := false
	walkNode(t.Tree.Root, func(n parse.Node) {
		if tn, ok := n.(*parse.TemplateNode); ok && tn.Name == "content" {
			found = true
		}
	})
	return found
}

// walkNode visits n and its children. text/template/parse exposes the node
// types but no walker; this covers the containers a layout can use.
func walkNode(n parse.Node, fn func(parse.Node)) {
	if n == nil {
		return
	}
	fn(n)
	switch n := n.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			walkNode(child, fn)
		}
	case *parse.ActionNode:
		walkNode(n.Pipe, fn)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			walkNode(cmd, fn)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			walkNode(arg, fn)
		}
	case *parse.ChainNode:
		walkNode(n.Node, fn)
	case *parse.IfNode:
		walkBranch(&n.BranchNode, fn)
	case *parse.RangeNode:
		walkBranch(&n.BranchNode, fn)
	case *parse.WithNode:
		walkBranch(&n.BranchNode, fn)
	case *parse.TemplateNode:
		walkNode(n.Pipe, fn)
	}
}

func walkBranch(b *parse.BranchNode, fn func(parse.Node)) {
	walkNode(b.Pipe, fn)
	walkNode(b.List, fn)
	walkNode(b.ElseList, fn)
}
