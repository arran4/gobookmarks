package gobookmarks

import "testing"

const tabBookmarkText = `Tab: One
Category: A
--
Tab: Two
Category: B
`

func TestExtractTabError(t *testing.T) {
	if _, err := ExtractTab(tabBookmarkText, "X"); err == nil {
		t.Fatalf("expected error")
	}
}
