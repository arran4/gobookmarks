package gobookmarks

import "testing"

const testBookmarkText = `Category: A
http://a.com a
Column
Category: B
http://b.com b
`

func TestExtractCategoryByIndexError(t *testing.T) {
	if _, err := ExtractCategoryByIndex(testBookmarkText, 5); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReplaceCategoryByIndexError(t *testing.T) {
	if _, err := ReplaceCategoryByIndex(testBookmarkText, 3, "foo"); err == nil {
		t.Fatalf("expected error")
	}
}
