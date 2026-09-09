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
	//go:embed "main.css" "logo.png"
	assetFS           embed.FS
	compiledTemplates *template.Template
	compileOnce       sync.Once

	assetRegistry     *Registry
	assetRegistryOnce sync.Once
)

func GetAssetRegistry() *Registry {
	assetRegistryOnce.Do(func() {
		var err error
		assetRegistry, err = NewRegistry(assetFS, false)
		if err != nil {
			panic(err)
		}
	})
	return assetRegistry
}

func GetAssetProvider() AssetProvider {
	return GetAssetRegistry()
}

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
	b, err := assetFS.ReadFile("main.css")
	if err != nil {
		panic(err)
	}
	return b
}

func GetFavicon() []byte {
	b, err := assetFS.ReadFile("logo.png")
	if err != nil {
		panic(err)
	}
	return b
}
