//go:build !live
// +build !live

package gobookmarks

import (
	"embed"
	"html/template"
)

var (
	//go:embed "templates/*.gohtml"
	templateFS embed.FS
	//go:embed "main.css"
	mainCSSData []byte
	//go:embed "logo.png"
	faviconData []byte
	//go:embed "static/js/*.js"
	staticJS embed.FS
)

func GetCompiledTemplates(funcs template.FuncMap) *template.Template {
	return template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.gohtml"))
}

func GetMainCSSData() []byte {
	return mainCSSData
}

func GetFavicon() []byte {
	return faviconData
}

func GetJS(name string) ([]byte, error) {
	return staticJS.ReadFile("static/js/" + name)
}
