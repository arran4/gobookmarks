package gobookmarks

import (
	"bytes"
	"embed"
	"io/fs"
	"path"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/txtar/*.txtar
var testdataFS embed.FS

func TestTxtarCases(t *testing.T) {
	entries, err := fs.Glob(testdataFS, "testdata/txtar/*.txtar")
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}

	for _, fixture := range entries {
		fixture := fixture
		t.Run(strings.TrimSuffix(path.Base(fixture), ".txtar"), func(t *testing.T) {
			raw, err := testdataFS.ReadFile(fixture)
			if err != nil {
				t.Fatalf("read fixture %s: %v", fixture, err)
			}
			ar := txtar.Parse(raw)

			var input, expected string
			for _, f := range ar.Files {
				if f.Name == "input.txt" {
					input = string(bytes.TrimSpace(f.Data))
				}
				if f.Name == "expected.txt" {
					expected = string(bytes.TrimSpace(f.Data))
				}
			}

			list := ParseBookmarks(input)
			got := strings.TrimSpace(list.String())

			if got != expected {
				t.Errorf("\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s\n", input, expected, got)
			}
		})
	}
}
