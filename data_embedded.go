//go:build !live

package gobookmarks

import (
	"embed"
	"html/template"
	"sync"
)

var (
	//go:embed all:templates
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
		// Also parse any .js templates since we now define bookmarks_parser.js as a template
		template.Must(ParseFSRecursive(compiledTemplates, templateFS, "templates", ".js"))
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
