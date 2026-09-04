package view

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestView_Render(t *testing.T) {
	var v *View
	var content template.HTML
	v = New()
	content = v.Render("html/tpl.html", map[string]interface{}{
		"Title":   "ThinkGo",
		"Message": "Hello ThinkGo !",
	})

	assert.Contains(t, content, "<h2>Hello ThinkGo !</h2>")
	assert.Contains(t, content, "<title>ThinkGo</title>")

	v = New()
	v.ParseGlob("html/*")
	content = v.Render("tpl.html", map[string]interface{}{
		"Title":   "ThinkGo",
		"Message": "Hello ThinkGo !",
	})

	assert.Contains(t, content, "<h2>Hello ThinkGo !</h2>")
	assert.Contains(t, content, "<title>ThinkGo</title>")
}
