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
