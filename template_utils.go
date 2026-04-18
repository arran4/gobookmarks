package gobookmarks

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
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
