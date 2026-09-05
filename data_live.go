//go:build live

package gobookmarks

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

func init() {
	log.Printf("Live data mode")
}

var (
	assetRegistry     *Registry
	assetRegistryOnce sync.Once
)

func getAssetDir() string {
	fsPath := "."
	if _, err := os.Stat(filepath.Join(fsPath, "main.css")); os.IsNotExist(err) {
		fsPath = "../../"
	}
	if _, err := os.Stat(filepath.Join(fsPath, "main.css")); os.IsNotExist(err) {
		fsPath = "../"
	}
	return fsPath
}

type LiveRegistryWrapper struct {
	reg *Registry
}

func (l *LiveRegistryWrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := l.reg.Reload(); err != nil {
		http.Error(w, "Internal Server Error: failed to reload assets", http.StatusInternalServerError)
		return
	}
	l.reg.ServeHTTP(w, req)
}

func (l *LiveRegistryWrapper) AssetURL(logicalName string) (string, error) {
	if err := l.reg.Reload(); err != nil {
		return "", err
	}
	return l.reg.AssetURL(logicalName)
}

func (l *LiveRegistryWrapper) GetAsset(url string) (*Asset, bool) {
	_ = l.reg.Reload() // GetAsset cannot return error in signature, best effort fallback to old state
	return l.reg.GetAsset(url)
}

func GetAssetRegistry() *Registry {
	assetRegistryOnce.Do(func() {
		dir := getAssetDir()
		var err error
		assetRegistry, err = NewRegistry(os.DirFS(dir), true)
		if err != nil {
			log.Printf("Asset registry error: %v", err)
		}
	})
	return assetRegistry
}

func GetAssetProvider() AssetProvider {
	return &LiveRegistryWrapper{reg: GetAssetRegistry()}
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
