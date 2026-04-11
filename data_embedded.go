//go:build !live
// +build !live

package gobookmarks

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

// ParseFSRecursive walks the given fs.FS starting at the base path and parses
// files matching the given extension into the template t. The template name
// will be the relative path from the base directory (e.g., "_partials/myform.gohtml"
// or "mainPage.gohtml").
func ParseFSRecursive(t *template.Template, fsys fs.FS, baseDir, ext string) (*template.Template, error) {
	err := fs.WalkDir(fsys, baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ext) {
			return nil
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		// ensure forward slashes for template names even on Windows
		name = filepath.ToSlash(name)
		_, err = t.New(name).Parse(string(b))
		return err
	})
	return t, err
}

var (
	//go:embed templates
	templateFS embed.FS
	//go:embed "main.css"
	mainCSSData []byte
	//go:embed "logo.png"
	faviconData []byte

	compiledTemplates *template.Template
	compileOnce       sync.Once
)

// GetCompiledTemplates returns a clone of the compiled templates with the given funcs applied.
// The templates are parsed only once at initialization using NewFuncs(nil) to establish the function map keys.
// The passed funcs (which should close over the request context) override the initial dummy functions.
func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	compileOnce.Do(func() {
		// Parse templates once. We use NewFuncs(nil) to provide the set of function names
		// required by the templates. The actual function implementations are irrelevant here
		// as they will be replaced by the request-specific funcs in the clone.
		t := template.New("").Funcs(NewFuncs(nil))
		compiledTemplates = template.Must(ParseFSRecursive(t, templateFS, "templates", ".gohtml"))
	})
	tmpl, err := compiledTemplates.Clone()
	if err != nil {
		panic(err)
	}
	return tmpl.Funcs(funcs)
}

func GetMainCSSData() []byte {
	return mainCSSData
}

func GetFavicon() []byte {
	return faviconData
}
