//go:build live
// +build live

package a4webbm

import (
	"html/template"
	"log"
	"os"
)

func init() {
	log.Printf("Live data mode")
}

func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	return template.Must(template.New("").Funcs(funcs).ParseFS(os.DirFS("./templates"), "*.gohtml"))
}

func GetMainCSSData() []byte {
	b, err := os.ReadFile("main.css")
	if err != nil {
		panic(err)
	}
	return b
}
