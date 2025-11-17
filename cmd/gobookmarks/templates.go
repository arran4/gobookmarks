package main

import (
	"embed"
	"sync"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	templates     *template.Template
	templatesOnce sync.Once
)

func GetTemplates() *template.Template {
	templatesOnce.Do(func() {
		templates = template.Must(template.ParseFS(templateFS, "templates/*.tmpl"))
	})
	return templates
}
