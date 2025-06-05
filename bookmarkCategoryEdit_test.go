package gobookmarks

import "testing"

const testBookmarkText = `Category: A
http://a.com a
Column
Category: B
http://b.com b
`

func TestExtractCategoryByIndex(t *testing.T) {
	got, err := ExtractCategoryByIndex(testBookmarkText, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Category: B\nhttp://b.com b\n"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestExtractCategoryByIndexFirst(t *testing.T) {
	got, err := ExtractCategoryByIndex(testBookmarkText, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Category: A\nhttp://a.com a"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestExtractCategoryByIndexError(t *testing.T) {
	if _, err := ExtractCategoryByIndex(testBookmarkText, 5); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReplaceCategoryByIndex(t *testing.T) {
	newSection := "Category: B\nhttp://new.com n"
	updated, err := ReplaceCategoryByIndex(testBookmarkText, 1, newSection)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Category: A\nhttp://a.com a\nColumn\n" + newSection
	if updated != expected {
		t.Fatalf("expected %q got %q", expected, updated)
	}
}

func TestReplaceCategoryByIndexFirst(t *testing.T) {
	newSection := "Category: A\nhttp://changed.com x"
	updated, err := ReplaceCategoryByIndex(testBookmarkText, 0, newSection)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := newSection + "\nColumn\nCategory: B\nhttp://b.com b\n"
	if updated != expected {
		t.Fatalf("expected %q got %q", expected, updated)
	}
}

func TestReplaceCategoryByIndexError(t *testing.T) {
	if _, err := ReplaceCategoryByIndex(testBookmarkText, 3, "foo"); err == nil {
		t.Fatalf("expected error")
	}
}
