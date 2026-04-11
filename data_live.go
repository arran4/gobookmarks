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
	_, callerFile, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(callerFile), "main.css"))
	if err != nil {
		panic(err)
	}
	return b
}

func GetFavicon() []byte {
	_, callerFile, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(callerFile), "logo.png"))
	if err != nil {
		panic(err)
	}
	return b
}
