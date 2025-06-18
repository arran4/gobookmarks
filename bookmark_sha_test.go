package gobookmarks

import (
	_ "embed"
	"testing"
)

var (
	//go:embed testdata/move_category_complex_input.txt
	shaComplex string
)

func TestPageShaStable(t *testing.T) {
	tabs := ParseBookmarks(shaComplex)
	sha1 := tabs[0].Pages[0].Sha()
	repro := ParseBookmarks(tabs.String())
	sha2 := repro[0].Pages[0].Sha()
	if sha1 != sha2 {
		t.Fatalf("sha mismatch")
	}
}
