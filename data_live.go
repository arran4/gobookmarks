//go:build live
// +build live

package main

import (
	"html/template"
	"os"
)

func getCompiledTemplates(funcs template.FuncMap) *template.Template {
	return template.Must(template.New("").Funcs(funcs).ParseFS(os.DirFS("./templates"), "*.gohtml"))
}

func getMainCSSData() []byte {
	b, err := os.ReadFile("main.css")
	if err != nil {
		panic(err)
	}
	return b
}
