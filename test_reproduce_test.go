package gobookmarks

import (
	"testing"
)

func TestExtractCategoryTabIssue(t *testing.T) {
	text := "Category: a\nhttps://example.com/ b\nTab: c\n"
	got, err := ExtractCategoryByIndex(text, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Category: a\nhttps://example.com/ b"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestReplaceCategoryTabIssue(t *testing.T) {
	text := "Category: a\nhttps://example.com/ b\nTab: c\n"
	updated, err := ReplaceCategoryByIndex(text, 0, "Category: z\nhttps://example.com/ x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Category: z\nhttps://example.com/ x\nTab: c\n"
	if updated != expected {
		t.Fatalf("expected %q got %q", expected, updated)
	}
}
