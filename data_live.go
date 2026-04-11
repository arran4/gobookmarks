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
	// When running `go test ./cmd/gobookmarks` the working dir is `./cmd/gobookmarks`
	// so `./templates` will not resolve.
	// But in normal `go build -tags live`, it runs from root. We need to handle both
	// or fallback, but a typical go test -tags live uses a path relative to where it runs.
	// the test command might be running from within cmd/gobookmarks where there is ALSO a "templates" directory
	// so we need to make sure we're getting the main web app templates which contain ".gohtml" files,
	// rather than the CLI templates (.gotmpl).
	fsPath := "./templates"
	if _, err := os.Stat(fsPath + "/mainPage.gohtml"); os.IsNotExist(err) {
		fsPath = "../../templates"
	}
	if _, err := os.Stat(fsPath + "/mainPage.gohtml"); os.IsNotExist(err) {
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
