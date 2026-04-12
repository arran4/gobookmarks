//go:build live
// +build live

package gobookmarks

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	log.Printf("Live data mode")
}

func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	t := template.New("").Funcs(funcs)
	// We use runtime.Caller to reliably find the project root from the location of this source file,
	// regardless of whether we are running from a deeper directory (e.g. `go test ./cmd/gobookmarks`).
	_, callerFile, _, ok := runtime.Caller(0)
	var fsPath string
	if ok {
		fsPath = filepath.Join(filepath.Dir(callerFile), "templates")
	} else {
		// Fallback just in case, though this should rarely happen.
		log.Printf("Warning: runtime.Caller(0) failed to resolve source file path, falling back to './templates'")
		fsPath = "./templates"
	}
	fsys := os.DirFS(fsPath)
	parsed, err := ParseFSRecursive(t, fsys, ".", ".gohtml")
	if err != nil {
		log.Printf("ParseFSRecursive error: %v", err)
	}
	return template.Must(parsed, err)
}

func GetMainCSSData() []byte {
	_, callerFile, _, ok := runtime.Caller(0)
	fsPath := "main.css"
	if ok {
		fsPath = filepath.Join(filepath.Dir(callerFile), fsPath)
	} else {
		log.Printf("Warning: runtime.Caller(0) failed to resolve source file path, falling back to './main.css'")
	}
	b, err := os.ReadFile(fsPath)
	if err != nil {
		panic(err)
	}
	return b
}

func GetFavicon() []byte {
	_, callerFile, _, ok := runtime.Caller(0)
	fsPath := "logo.png"
	if ok {
		fsPath = filepath.Join(filepath.Dir(callerFile), fsPath)
	} else {
		log.Printf("Warning: runtime.Caller(0) failed to resolve source file path, falling back to './logo.png'")
	}
	b, err := os.ReadFile(fsPath)
	if err != nil {
		panic(err)
	}
	return b
}
