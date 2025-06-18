package gobookmarks

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestSerializeBookmarksRoundTrip(t *testing.T) {
	samples := []string{
		defaultBookmarks,
		complexBookmarkText,
		multiBookmarkText,
	}
	for _, in := range samples {
		tabs1 := PreprocessBookmarks(in)
		out := SerializeBookmarks(tabs1)
		tabs2 := PreprocessBookmarks(out)
		if diff := cmp.Diff(tabs1, tabs2); diff != "" {
			t.Fatalf("round trip diff:\n%s", diff)
		}
	}
}
