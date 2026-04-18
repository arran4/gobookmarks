//go:build live

package gobookmarks

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
)

func init() {
	log.Printf("Live data mode")
}

func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	t := template.New("").Funcs(funcs)
	// When running `go test ./cmd/gobookmarks` the working dir is `./cmd/gobookmarks`
	// so `./templates` might resolve to the CLI templates directory instead of the main one.
	// We specifically look for mainPage.gohtml to ensure we found the web application templates.
	fsPath := "./templates"
	if _, err := os.Stat(filepath.Join(fsPath, "mainPage.gohtml")); os.IsNotExist(err) {
		fsPath = "../../templates"
	}
	if _, err := os.Stat(filepath.Join(fsPath, "mainPage.gohtml")); os.IsNotExist(err) {
		fsPath = "../templates"
	}
	fsys := os.DirFS(fsPath)
	parsed, err := ParseFSRecursive(t, fsys, ".", ".gohtml")
	if err != nil {
		log.Printf("ParseFSRecursive error: %v", err)
	}
	return template.Must(parsed, err)
}

func GetMainCSSData() []byte {
	fsPath := "main.css"
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../../main.css"
	}
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../main.css"
	}
	b, err := os.ReadFile(fsPath)
	if err != nil {
		panic(err)
	}
	return b
}

func GetFavicon() []byte {
	fsPath := "logo.png"
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../../logo.png"
	}
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../logo.png"
	}
	b, err := os.ReadFile(fsPath)
	if err != nil {
		panic(err)
	}
	return b
}
