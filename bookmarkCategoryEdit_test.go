package gobookmarks

import (
	"strings"
	"testing"
)

func TestExtractCategoryByIndex_Boundaries(t *testing.T) {
	input := "Column: First\nCategory: Alpha\nColumn\nCategory: Beta\nColumn: Third\nCategory: Gamma\nColumn:   "

	res, err := ExtractCategoryByIndex(input, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res) != "Category: Alpha" {
		t.Errorf("Expected 'Category: Alpha', got %q", res)
	}

	res, err = ExtractCategoryByIndex(input, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res) != "Category: Beta" {
		t.Errorf("Expected 'Category: Beta', got %q", res)
	}

	res, err = ExtractCategoryByIndex(input, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res) != "Category: Gamma" {
		t.Errorf("Expected 'Category: Gamma', got %q", res)
	}
}

func TestReplaceCategoryByIndex_Boundaries(t *testing.T) {
	input := "Column: First\nCategory: Alpha\nColumn\nCategory: Beta\nColumn: Third\nCategory: Gamma\n"

	res, err := ReplaceCategoryByIndex(input, 1, "Category: Replaced")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Column: First\nCategory: Alpha\nColumn\nCategory: Replaced\nColumn: Third\nCategory: Gamma\n"
	if res != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, res)
	}
}
