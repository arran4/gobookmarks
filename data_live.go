//go:build live
// +build live

package gobookmarks

import (
	"html/template"
	"log"
	"os"
)

func init() {
	log.Printf("Live data mode")
}

func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	t := template.New("").Funcs(funcs)
	return template.Must(ParseFSRecursive(t, os.DirFS("./templates"), ".", ".gohtml"))
}

func GetMainCSSData() []byte {
	b, err := os.ReadFile("main.css")
	if err != nil {
		panic(err)
	}
	return b
}

func GetFavicon() []byte {
	b, err := os.ReadFile("logo.png")
	if err != nil {
		panic(err)
	}
	return b
}
