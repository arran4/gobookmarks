package gobookmarks

import (
	"strings"
	"testing"
)

func TestTabOperationsImplicit(t *testing.T) {
	input := "Category: A\nTab: Named\nCategory: B\n"
	tabs := PreprocessBookmarks(input)
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs got %d", len(tabs))
	}
	tabs = insertTab(tabs, 1)
	if len(tabs) != 3 {
		t.Fatalf("expected 3 tabs after insert got %d", len(tabs))
	}
	tabs = moveTab(tabs, 2, -2)
	out := SerializeBookmarks(tabs)
	round := PreprocessBookmarks(out)
	if len(round) != 3 {
		t.Fatalf("expected 3 tabs after round trip got %d", len(round))
	}
	if round[0].Name != "Named" {
		t.Fatalf("expected first tab Named got %q", round[0].Name)
	}
	if round[1].Name != "" {
		t.Fatalf("expected second tab unnamed got %q", round[1].Name)
	}
	round = deleteTab(round, 1)
	if len(round) != 2 {
		t.Fatalf("expected 2 tabs after delete got %d", len(round))
	}
}

func TestMoveImplicitMainTabDown(t *testing.T) {
	input := "Category:1\nhttp://a a\n\nTab: Tab2\nCategory\nhttp://a a"
	tabs := PreprocessBookmarks(input)
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs")
	}
	tabs = moveTab(tabs, 0, 1)
	out := SerializeBookmarks(tabs)
	expected := "Tab: Tab2\nCategory: Category\nhttp://a a\nTab:\nCategory: 1\nhttp://a a"
	if strings.TrimSpace(out) != expected {
		t.Fatalf("unexpected output:\n%s", out)
	}
	round := PreprocessBookmarks(out)
	if round[0].Name != "Tab2" || round[1].Name != "" {
		t.Fatalf("tab names wrong: %#v %#v", round[0].Name, round[1].Name)
	}
}

func TestPageOperations(t *testing.T) {
	input := "Category: A\nPage: B\nCategory: C\n"
	tabs := PreprocessBookmarks(input)
	t0 := tabs[0]
	insertPage(t0, 1)
	if len(t0.Pages) != 3 {
		t.Fatalf("expected 3 pages after insert got %d", len(t0.Pages))
	}
	movePage(t0, 2, -2)
	out := SerializeBookmarks(tabs)
	round := PreprocessBookmarks(out)
	if round[0].Pages[0].Name != "B" {
		t.Fatalf("expected first page B got %q", round[0].Pages[0].Name)
	}
	deletePage(round[0], 1)
	if len(round[0].Pages) != 2 {
		t.Fatalf("expected 2 pages after delete got %d", len(round[0].Pages))
	}
}

func TestCategoryOperations(t *testing.T) {
	input := "Category: A\nCategory: B\n"
	tabs := PreprocessBookmarks(input)
	t0 := tabs[0]
	moveCategory(t0, 0, 0, 0, 1, -1)
	insertCategory(t0, 0, 0, 0, 1)
	cats := t0.Pages[0].Blocks[0].Columns[0].Categories
	if len(cats) != 3 {
		t.Fatalf("expected 3 categories after insert got %d", len(cats))
	}
	if cats[0].Name != "B" {
		t.Fatalf("expected first category B got %q", cats[0].Name)
	}
	deleteCategory(t0, 0, 0, 0, 1)
	if len(cats)-1 != 2 {
		t.Fatalf("expected 2 categories after delete got %d", len(cats)-1)
	}
}

func TestEntryOperations(t *testing.T) {
	input := "Category: C\nhttps://a A\nhttps://b B\n"
	tabs := PreprocessBookmarks(input)
	cat := tabs[0].Pages[0].Blocks[0].Columns[0].Categories[0]
	insertEntry(cat, 1)
	if len(cat.Entries) != 3 {
		t.Fatalf("expected 3 entries after insert got %d", len(cat.Entries))
	}
	moveEntry(cat, 2, -2)
	if cat.Entries[0].Url != "https://b" {
		t.Fatalf("expected first entry https://b got %q", cat.Entries[0].Url)
	}
	deleteEntry(cat, 1)
	if len(cat.Entries) != 2 {
		t.Fatalf("expected 2 entries after delete got %d", len(cat.Entries))
	}
}

func TestInsertColumnAndMoveAcross(t *testing.T) {
	input := "Category: A\nColumn\nCategory: B\n"
	tabs := PreprocessBookmarks(input)
	p := tabs[0].Pages[0]
	insertColumn(p, 0, 1)
	if len(p.Blocks[0].Columns) != 3 {
		t.Fatalf("expected 3 columns got %d", len(p.Blocks[0].Columns))
	}
	moveCategoryTo(tabs[0], 0, 0, 0, 0, 0, 0, 2, 0)
	if len(p.Blocks[0].Columns[0].Categories) != 0 || len(p.Blocks[0].Columns[2].Categories) != 2 {
		t.Fatalf("moveCategoryTo failed")
	}
	if p.Blocks[0].Columns[2].Categories[0].Name != "A" {
		t.Fatalf("expected moved category A")
	}
}

func TestMoveEntryBetween(t *testing.T) {
	input := "Category: A\nhttps://a A\nCategory: B\nhttps://b B\n"
	tabs := PreprocessBookmarks(input)
	cats := tabs[0].Pages[0].Blocks[0].Columns[0].Categories
	moveEntryBetween(cats[0], 0, cats[1], 1)
	if len(cats[0].Entries) != 0 || len(cats[1].Entries) != 2 {
		t.Fatalf("moveEntryBetween counts wrong")
	}
	if cats[1].Entries[1].Url != "https://a" {
		t.Fatalf("entry not moved")
	}
}
