package view

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-think/flow"
)

// the default View.
var defaultView = New()

type View struct {
	tmpl     *template.Template
	viewsDir string
	funcs    template.FuncMap
}

func New() *View {
	return &View{
		viewsDir: "resources/views",
		funcs:    make(template.FuncMap),
	}
}

// SetViewsDir sets the base directory for views templates.
func (v *View) SetViewsDir(dir string) *View {
	v.viewsDir = dir
	return v
}

// Funcs registers custom template helper functions.
func (v *View) Funcs(funcMap template.FuncMap) *View {
	for name, fn := range funcMap {
		v.funcs[name] = fn
	}
	return v
}

// ParseGlob creates a new Template and parses the template definitions from the
// files identified by the pattern.
func (v *View) ParseGlob(pattern string) {
	if pattern == "" {
		return
	}

	if fi, err := os.Stat(pattern); err == nil && fi.IsDir() {
		pattern = filepath.Join(pattern, "*")
	} else if !strings.Contains(pattern, "*") {
		pattern = pattern + "/*"
	}

	t, err := template.ParseGlob(pattern)
	if err == nil {
		v.tmpl = t
	}
}

func (v *View) defaultFuncMap(req *flow.Request) template.FuncMap {
	fm := template.FuncMap{
		"raw": func(s string) template.HTML {
			return template.HTML(s)
		},
		"csrf_token": func() string {
			if req != nil && req.Session() != nil {
				return req.Session().Token()
			}
			return ""
		},
		"csrf_field": func() template.HTML {
			if req != nil && req.Session() != nil {
				tok := req.Session().Token()
				return template.HTML(fmt.Sprintf(`<input type="hidden" name="_token" value="%s">`, template.HTMLEscapeString(tok)))
			}
			return ""
		},
		"old": func(key string, defaultValue ...string) string {
			if req != nil {
				return req.Old(key, defaultValue...)
			}
			if len(defaultValue) > 0 {
				return defaultValue[0]
			}
			return ""
		},
		"has_error": func(key string) bool {
			if req != nil && req.Session() != nil {
				raw := req.Session().Get("errors")
				if m, ok := raw.(map[string][]string); ok {
					return len(m[key]) > 0
				}
				if m, ok := raw.(map[string]any); ok {
					_, exists := m[key]
					return exists
				}
				if m, ok := raw.(map[string]string); ok {
					_, exists := m[key]
					return exists
				}
			}
			return false
		},
		"error": func(key string) string {
			if req != nil && req.Session() != nil {
				raw := req.Session().Get("errors")
				if m, ok := raw.(map[string][]string); ok {
					if len(m[key]) > 0 {
						return m[key][0]
					}
				}
				if m, ok := raw.(map[string]string); ok {
					return m[key]
				}
				if m, ok := raw.(map[string]any); ok {
					if errList, ok2 := m[key].([]string); ok2 && len(errList) > 0 {
						return errList[0]
					}
					if errList, ok2 := m[key].([]any); ok2 && len(errList) > 0 {
						return fmt.Sprintf("%v", errList[0])
					}
					return fmt.Sprintf("%v", m[key])
				}
			}
			return ""
		},
		"errors": func(key ...string) []string {
			if req != nil && req.Session() != nil {
				raw := req.Session().Get("errors")
				if m, ok := raw.(map[string][]string); ok {
					if len(key) > 0 {
						return m[key[0]]
					}
					var all []string
					for _, errs := range m {
						all = append(all, errs...)
					}
					return all
				}
			}
			return nil
		},
		"auth_check": func() bool {
			if req != nil && req.Session() != nil {
				return req.Session().Has("user_id")
			}
			return false
		},
		"auth_user": func(field ...string) any {
			if req != nil && req.Session() != nil {
				if len(field) > 0 {
					return req.Session().Get("user_" + field[0])
				}
				name := req.Session().Get("user_name")
				if name != nil {
					return name
				}
				return req.Session().Get("user_id")
			}
			return ""
		},
		"session": func(key string, def ...any) any {
			if req != nil && req.Session() != nil {
				if val := req.Session().Get(key); val != nil {
					return val
				}
			}
			if len(def) > 0 {
				return def[0]
			}
			return ""
		},
	}

	for k, fn := range v.funcs {
		fm[k] = fn
	}
	return fm
}

func (v *View) resolveFile(name string) string {
	candidates := []string{
		name,
		name + ".html",
		filepath.Join(v.viewsDir, name),
		filepath.Join(v.viewsDir, name+".html"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

// Render renders a template by name with given data and optional request context.
func (v *View) Render(name string, data any, req ...*flow.Request) template.HTML {
	var r *flow.Request
	if len(req) > 0 {
		r = req[0]
	}

	// 1. If v.tmpl was populated via ParseGlob, use it directly (backward compatibility)
	if v.tmpl != nil {
		var buf bytes.Buffer
		tplName := path.Base(name)
		err := v.tmpl.ExecuteTemplate(&buf, tplName, data)
		if err == nil {
			return template.HTML(buf.String())
		}
	}

	// 2. Resolve content template file
	contentPath := v.resolveFile(name)
	if contentPath == "" {
		// Fallback to directory glob
		pattern := path.Join(path.Dir(name), "*")
		t, err := template.New("").Funcs(v.defaultFuncMap(r)).ParseGlob(pattern)
		if err == nil {
			var buf bytes.Buffer
			_ = t.ExecuteTemplate(&buf, path.Base(name), data)
			return template.HTML(buf.String())
		}
		return ""
	}

	// 3. Check for layout specified in data map (e.g. data["Layout"] = "layouts/guest")
	var layoutName string
	if m, ok := data.(map[string]any); ok {
		if l, ok2 := m["Layout"].(string); ok2 && l != "" {
			layoutName = l
		}
	} else if m, ok := data.(map[string]string); ok {
		if l, ok2 := m["Layout"]; ok2 && l != "" {
			layoutName = l
		}
	}

	if layoutName != "" {
		return v.RenderWithLayout(layoutName, name, data, req...)
	}

	// 4. Try parsing directory glob first so shared definitions (like layout) work
	dir := filepath.Dir(contentPath)
	t, err := template.New("").Funcs(v.defaultFuncMap(r)).ParseGlob(filepath.Join(dir, "*"))
	if err == nil {
		var buf bytes.Buffer
		if err2 := t.ExecuteTemplate(&buf, filepath.Base(contentPath), data); err2 == nil {
			return template.HTML(buf.String())
		}
	}

	// 5. Standalone template execution fallback
	tmpl, err := template.New(filepath.Base(contentPath)).
		Funcs(v.defaultFuncMap(r)).
		ParseFiles(contentPath)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, data)
	return template.HTML(buf.String())
}

// RenderWithLayout renders a content view template embedded inside a layout template.
func (v *View) RenderWithLayout(layout, name string, data any, req ...*flow.Request) template.HTML {
	var r *flow.Request
	if len(req) > 0 {
		r = req[0]
	}

	layoutPath := v.resolveFile(layout)
	contentPath := v.resolveFile(name)

	if layoutPath == "" || contentPath == "" {
		if contentPath != "" {
			return v.Render(name, data, req...)
		}
		return ""
	}

	tmpl, err := template.New(filepath.Base(layoutPath)).
		Funcs(v.defaultFuncMap(r)).
		ParseFiles(layoutPath, contentPath)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		// Try executing "layout" or "content" if root execution fails
		_ = tmpl.ExecuteTemplate(&buf, "layout", data)
	}

	return template.HTML(buf.String())
}

// HTML returns a *flow.Response with content-type text/html.
func (v *View) HTML(name string, data any, req ...*flow.Request) *flow.Response {
	content := v.Render(name, data, req...)
	return flow.Html(string(content))
}

// HTMLWithLayout returns a *flow.Response rendered with a specific layout.
func (v *View) HTMLWithLayout(layout, name string, data any, req ...*flow.Request) *flow.Response {
	content := v.RenderWithLayout(layout, name, data, req...)
	return flow.Html(string(content))
}

// Package-level functions delegating to defaultView
func Render(name string, data any, req ...*flow.Request) template.HTML {
	return defaultView.Render(name, data, req...)
}

func RenderWithLayout(layout, name string, data any, req ...*flow.Request) template.HTML {
	return defaultView.RenderWithLayout(layout, name, data, req...)
}

func HTML(name string, data any, req ...*flow.Request) *flow.Response {
	return defaultView.HTML(name, data, req...)
}

func HTMLWithLayout(layout, name string, data any, req ...*flow.Request) *flow.Response {
	return defaultView.HTMLWithLayout(layout, name, data, req...)
}

func ParseGlob(pattern string) {
	defaultView.ParseGlob(pattern)
}

func SetViewsDir(dir string) {
	defaultView.SetViewsDir(dir)
}

func Funcs(funcMap template.FuncMap) {
	defaultView.Funcs(funcMap)
}
