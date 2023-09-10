//go:build !live
// +build !live

package main

import (
	"embed"
	"html/template"
)

var (
	//go:embed "templates/*.gohtml"
	templateFS embed.FS
	//go:embed "main.css"
	mainCSSData []byte
)

func getCompiledTemplates(funcs template.FuncMap) *template.Template {
	return template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.gohtml"))
}

func getMainCSSData() []byte {
	return mainCSSData
}
