package gobookmarks

import (
	"embed"
	"strings"
	"testing"
)

var _ embed.FS

//go:embed testdata/add_page_input.txt
var addPageInput string

//go:embed testdata/add_page_expected.txt
var addPageExpected string

func TestAddPageBusiness(t *testing.T) {
	tabs := PreprocessBookmarks(strings.TrimSpace(addPageInput))
	insertPage(tabs[0], 1)
	tabs[0].Pages[1].Name = "New"
	got := SerializeBookmarks(tabs)
	if got != strings.TrimSpace(addPageExpected) {
		t.Fatalf("unexpected bookmarks: %q", got)
	}
}
