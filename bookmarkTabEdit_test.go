package gobookmarks

import "testing"

const tabBookmarkText = `Tab: One
Category: A
--
Tab: Two
Category: B
`

func TestExtractTab(t *testing.T) {
	got, err := ExtractTab(tabBookmarkText, "Two")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	expected := "Tab: Two\nCategory: B\n"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestExtractTabError(t *testing.T) {
	if _, err := ExtractTab(tabBookmarkText, "X"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReplaceTab(t *testing.T) {
	updated, err := ReplaceTab(tabBookmarkText, "Two", "Z", "Category: C")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	expected := "Tab: One\nCategory: A\n--\nTab: Z\nCategory: C"
	if updated != expected {
		t.Fatalf("expected %q got %q", expected, updated)
	}
}

func TestAppendTab(t *testing.T) {
	updated := AppendTab("Category: X", "New", "Category: Y")
	expected := "Category: X\nTab: New\nCategory: Y\n"
	if updated != expected {
		t.Fatalf("expected %q got %q", expected, updated)
	}
}
