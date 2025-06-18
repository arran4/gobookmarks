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
)

func TestMoveCategory(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	if err := tabs.MoveCategory(4, 1, false); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	if got != moveBeforeExpected {
		t.Fatalf("expected %q got %q", moveBeforeExpected, got)
	}
}

func TestMoveCategoryNewColumn(t *testing.T) {
	tabs := ParseBookmarks(moveComplexInput)
	if err := tabs.MoveCategory(0, -1, true); err != nil {
		t.Fatalf("MoveCategory: %v", err)
	}
	got := tabs.String()
	if got != moveNewColumnExpected {
		t.Fatalf("expected %q got %q", moveNewColumnExpected, got)
	}
}
