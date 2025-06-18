package gobookmarks

import (
	_ "embed"
	"testing"
)

var (
	//go:embed testdata/insert_category_input.txt
	insertCategoryInput string
	//go:embed testdata/insert_category_expected.txt
	insertCategoryExpected string

	//go:embed testdata/add_category_input.txt
	addCategoryInput string
	//go:embed testdata/add_category_expected.txt
	addCategoryExpected string

	//go:embed testdata/switch_category_input.txt
	switchCategoryInput string
	//go:embed testdata/switch_category_expected.txt
	switchCategoryExpected string

	//go:embed testdata/add_page_input.txt
	addPageInput string
	//go:embed testdata/add_page_expected.txt
	addPageExpected string

	//go:embed testdata/insert_page_input.txt
	insertPageInput string
	//go:embed testdata/insert_page_expected.txt
	insertPageExpected string

	//go:embed testdata/switch_page_input.txt
	switchPageInput string
	//go:embed testdata/switch_page_expected.txt
	switchPageExpected string

	//go:embed testdata/add_tab_input.txt
	addTabInput string
	//go:embed testdata/add_tab_expected.txt
	addTabExpected string

	//go:embed testdata/insert_tab_input.txt
	insertTabInput string
	//go:embed testdata/insert_tab_expected.txt
	insertTabExpected string

	//go:embed testdata/switch_tab_input.txt
	switchTabInput string
	//go:embed testdata/switch_tab_expected.txt
	switchTabExpected string
)

func TestInsertCategory(t *testing.T) {
	tabs := ParseBookmarks(insertCategoryInput)
	col := tabs[0].Pages[0].Blocks[0].Columns[0]
	col.InsertCategory(1, &BookmarkCategory{Name: "C"})
	got := tabs.String()
	if got != insertCategoryExpected {
		t.Fatalf("expected %q got %q", insertCategoryExpected, got)
	}
}

func TestAddCategory(t *testing.T) {
	tabs := ParseBookmarks(addCategoryInput)
	col := tabs[0].Pages[0].Blocks[0].Columns[0]
	col.AddCategory(&BookmarkCategory{Name: "B"})
	got := tabs.String()
	if got != addCategoryExpected {
		t.Fatalf("expected %q got %q", addCategoryExpected, got)
	}
}

func TestSwitchCategory(t *testing.T) {
	tabs := ParseBookmarks(switchCategoryInput)
	col := tabs[0].Pages[0].Blocks[0].Columns[0]
	col.SwitchCategories(0, 1)
	got := tabs.String()
	if got != switchCategoryExpected {
		t.Fatalf("expected %q got %q", switchCategoryExpected, got)
	}
}

func TestAddPage(t *testing.T) {
	tabs := ParseBookmarks(addPageInput)
	p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
	p.Blocks[0].Columns[0].AddCategory(&BookmarkCategory{Name: "B"})
	tabs[0].AddPage(p)
	got := tabs.String()
	if got != addPageExpected {
		t.Fatalf("expected %q got %q", addPageExpected, got)
	}
}

func TestInsertPage(t *testing.T) {
	tabs := ParseBookmarks(insertPageInput)
	p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
	p.Blocks[0].Columns[0].AddCategory(&BookmarkCategory{Name: "X"})
	tabs[0].InsertPage(1, p)
	got := tabs.String()
	if got != insertPageExpected {
		t.Fatalf("expected %q got %q", insertPageExpected, got)
	}
}

func TestSwitchPage(t *testing.T) {
	tabs := ParseBookmarks(switchPageInput)
	tabs[0].SwitchPages(0, 1)
	got := tabs.String()
	if got != switchPageExpected {
		t.Fatalf("expected %q got %q", switchPageExpected, got)
	}
}

func TestAddTab(t *testing.T) {
	tabs := ParseBookmarks(addTabInput)
	nl := &BookmarkTab{Name: "Three"}
	p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
	p.Blocks[0].Columns[0].AddCategory(&BookmarkCategory{Name: "C"})
	nl.AddPage(p)
	var list BookmarkList = tabs
	list.AddTab(nl)
	got := list.String()
	if got != addTabExpected {
		t.Fatalf("expected %q got %q", addTabExpected, got)
	}
}

func TestInsertTab(t *testing.T) {
	tabs := ParseBookmarks(insertTabInput)
	nl := &BookmarkTab{Name: "Mid"}
	p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
	p.Blocks[0].Columns[0].AddCategory(&BookmarkCategory{Name: "X"})
	nl.AddPage(p)
	var list BookmarkList = tabs
	list.InsertTab(1, nl)
	got := list.String()
	if got != insertTabExpected {
		t.Fatalf("expected %q got %q", insertTabExpected, got)
	}
}

func TestSwitchTab(t *testing.T) {
	tabs := ParseBookmarks(switchTabInput)
	var list BookmarkList = tabs
	list.SwitchTabs(0, 1)
	got := list.String()
	if got != switchTabExpected {
		t.Fatalf("expected %q got %q", switchTabExpected, got)
	}
}

func TestMoveTab(t *testing.T) {
	tabs := ParseBookmarks(switchTabInput)
	var list BookmarkList = tabs
	list.MoveTab(0, 1)
	got := list.String()
	if got != switchTabExpected {
		t.Fatalf("expected %q got %q", switchTabExpected, got)
	}
}

func TestMovePage(t *testing.T) {
	tabs := ParseBookmarks(switchPageInput)
	tabs[0].MovePage(0, 1)
	got := tabs.String()
	if got != switchPageExpected {
		t.Fatalf("expected %q got %q", switchPageExpected, got)
	}
}

func TestMoveEntry(t *testing.T) {
	cat := &BookmarkCategory{Name: "C", Entries: []*BookmarkEntry{{Url: "u1"}, {Url: "u2"}}}
	cat.MoveEntry(0, 1)
	expected := "Category: C\nu2\nu1\n"
	if got := cat.String(); got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestInvalidOperations(t *testing.T) {
	tabs := ParseBookmarks(insertCategoryInput)
	col := tabs[0].Pages[0].Blocks[0].Columns[0]
	orig := len(col.Categories)
	col.InsertCategory(-1, &BookmarkCategory{Name: "X"})
	if len(col.Categories) != orig {
		t.Fatalf("invalid insert changed categories")
	}
	col.SwitchCategories(-1, 3)
	if len(col.Categories) != orig {
		t.Fatalf("invalid switch changed categories")
	}

	tabs = ParseBookmarks(insertPageInput)
	pcount := len(tabs[0].Pages)
	tabs[0].InsertPage(-1, &BookmarkPage{})
	if len(tabs[0].Pages) != pcount {
		t.Fatalf("invalid page insert changed pages")
	}
	tabs[0].SwitchPages(-1, 5)
	if len(tabs[0].Pages) != pcount {
		t.Fatalf("invalid page switch changed pages")
	}

	tabs = ParseBookmarks(insertTabInput)
	l := BookmarkList(tabs)
	tcount := len(l)
	l.InsertTab(-1, &BookmarkTab{})
	if len(l) != tcount {
		t.Fatalf("invalid tab insert changed tabs")
	}
	l.SwitchTabs(-1, 9)
	if len(l) != tcount {
		t.Fatalf("invalid tab switch changed tabs")
	}
}

func TestParseEmpty(t *testing.T) {
	tabs := ParseBookmarks("")
	if len(tabs) != 1 || len(tabs[0].Pages) != 1 {
		t.Fatalf("expected single empty tab and page")
	}
	if got := tabs.String(); got != "" {
		t.Fatalf("expected empty serialization got %q", got)
	}
}

func TestStringers(t *testing.T) {
	e := &BookmarkEntry{Url: "u", Name: "n"}
	if e.String() != "u n\n" {
		t.Fatalf("entry string")
	}
	e2 := &BookmarkEntry{Url: "u"}
	if e2.String() != "u\n" {
		t.Fatalf("entry fallback")
	}
	var nilEntry *BookmarkEntry
	if nilEntry.String() != "" {
		t.Fatalf("nil entry")
	}

	c := &BookmarkCategory{Name: "C", Entries: []*BookmarkEntry{e}}
	if c.String() != "Category: C\nu n\n" {
		t.Fatalf("category string")
	}

	col := &BookmarkColumn{Categories: []*BookmarkCategory{c}}
	if col.String() != "Category: C\nu n\n" {
		t.Fatalf("column string")
	}

	blk := &BookmarkBlock{Columns: []*BookmarkColumn{col, {}}}
	blk.Columns[1].AddCategory(&BookmarkCategory{Name: "D"})
	expectedBlk := "Category: C\nu n\nColumn\nCategory: D\n"
	if blk.String() != expectedBlk {
		t.Fatalf("block string")
	}

	page := &BookmarkPage{Name: "First", Blocks: []*BookmarkBlock{blk, {HR: true}}}
	expectedPage := expectedBlk + "--\n"
	if page.String() != expectedPage {
		t.Fatalf("page string")
	}

	tab := &BookmarkTab{Name: "T", Pages: []*BookmarkPage{page}}
	expectTab := "Tab: T\nPage: First\n" + expectedPage
	if tab.String() != expectTab {
		t.Fatalf("tab string")
	}

	anon := &BookmarkTab{Pages: []*BookmarkPage{page}}
	if anon.String() != "Tab\nPage: First\n"+expectedPage {
		t.Fatalf("anon tab string")
	}

	list := BookmarkList{tab}
	if list.String() != expectTab {
		t.Fatalf("list string")
	}

	// coverage of additional branches
	page2 := &BookmarkPage{Name: "N2"}
	tab2 := &BookmarkTab{Name: "X", Pages: []*BookmarkPage{page, page2}}
	full := BookmarkList{tab2}
	want := "Tab: X\nPage: First\n" + expectedPage + "Page: N2\n"
	if full.String() != want {
		t.Fatalf("full list string")
	}

	if tab.stringWithContext(true) != expectTab {
		t.Fatalf("stringWithContext named")
	}
	if anon.stringWithContext(false) != "Tab\nPage: First\n"+expectedPage {
		t.Fatalf("stringWithContext anon")
	}
	if anon.stringWithContext(true) != "Page: First\n"+expectedPage {
		t.Fatalf("stringWithContext omit")
	}
}
