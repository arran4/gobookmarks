package gobookmarks

import (
	_ "embed"
	"testing"
)

var (
	//go:embed testdata/move_category_complex_input.txt
	moveComplexInput string
	//go:embed testdata/move_category_before_expected.txt
	moveBeforeExpected string
	//go:embed testdata/move_category_newcolumn_expected.txt
	moveNewColumnExpected string
	//go:embed testdata/move_category_end_expected.txt
	moveEndExpected string
	//go:embed testdata/move_category_end_lastpage_expected.txt
	moveEndLastPageExpected string
)

func TestMoveCategory(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	if err := tabs.MoveCategoryBefore(4, 1); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	if got != moveBeforeExpected {
		t.Fatalf("expected %q got %q", moveBeforeExpected, got)
	}
}

func TestMoveCategoryNewColumn(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	if err := tabs.MoveCategoryNewColumn(0, tabs[1].Pages[0], -1); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	if got != moveNewColumnExpected {
		t.Fatalf("expected %q got %q", moveNewColumnExpected, got)
	}
}

func TestMoveCategoryEndColumn(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	if err := tabs.MoveCategoryToEnd(0, tabs[0].Pages[0], 1); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	if got != moveEndExpected {
		t.Fatalf("expected %q got %q", moveEndExpected, got)
	}
}

func TestMoveCategoryEndLastPage(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	destPage := tabs[0].Pages[len(tabs[0].Pages)-1]
	lastBlock := destPage.Blocks[len(destPage.Blocks)-1]
	destCol := len(lastBlock.Columns) - 1
	if err := tabs.MoveCategoryToEnd(0, destPage, destCol); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	expected := moveEndLastPageExpected
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestMoveCategory_PreservesNamedEmptyColumn(t *testing.T) {
	input := "Tab: A\nPage: B\nColumn: Old\nCategory: X\n"
	tabs := ParseBookmarks(input)

	if err := tabs.MoveCategoryNewColumn(0, tabs[0].Pages[0], 0); err != nil {
		t.Fatalf("MoveCategoryNewColumn: %v", err)
	}

	got := tabs.String()
	expected := "Tab: A\nPage: B\nColumn: Old\nColumn\nCategory: X\n"
	if got != expected {
		t.Fatalf("expected\n%q\ngot\n%q", expected, got)
	}
}

func TestMoveCategory_RemovesUnnamedEmptyColumn(t *testing.T) {
	input := "Tab: A\nPage: B\nCategory: X\n"
	tabs := ParseBookmarks(input)

	if err := tabs.MoveCategoryNewColumn(0, tabs[0].Pages[0], 0); err != nil {
		t.Fatalf("MoveCategoryNewColumn: %v", err)
	}

	got := tabs.String()
	expected := "Tab: A\nPage: B\nCategory: X\n"
	if got != expected {
		t.Fatalf("expected\n%q\ngot\n%q", expected, got)
	}
}
