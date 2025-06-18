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
		tabs1 := ParseBookmarks(in)
		out := tabs1.String()
		tabs2 := ParseBookmarks(out)
		if diff := cmp.Diff(tabs1, tabs2); diff != "" {
			t.Fatalf("round trip diff:\n%s", diff)
		}
	}
}
