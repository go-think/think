package view

import (
	"bytes"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// the default View.
var view = &View{}

type View struct {
	tmpl *template.Template
}

func New() *View {
	v := &View{}
	return v
}

// ParseGlob creates a new Template and parses the template definitions from the
// files identified by the pattern.
func (v *View) ParseGlob(pattern string) {
	if pattern == "" {
		return
	}

	// If the provided path is a directory, automatically add wildcard
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

func (v *View) Render(name string, data interface{}) template.HTML {
	tmpl := v.tmpl
	if tmpl == nil {
		pattern := path.Join(path.Dir(name), "*")
		name = path.Base(name)
		t, err := template.ParseGlob(pattern)
		if err == nil {
			tmpl = t
		}
	}

	if tmpl == nil {
		return ""
	}

	var buf bytes.Buffer
	_ = tmpl.ExecuteTemplate(&buf, name, data)

	return template.HTML(buf.String())
}

func Render(name string, data interface{}) template.HTML {
	return view.Render(name, data)
}

func ParseGlob(pattern string) {
	view.ParseGlob(pattern)
}
