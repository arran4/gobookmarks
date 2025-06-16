package gobookmarks

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
)

type (
	Pg  = BookmarkPage
	Blk = BookmarkBlock
	Col = BookmarkColumn
	Cat = BookmarkCategory
	Ent = BookmarkEntry
)

func e(u, n string) *Ent                  { return &Ent{Url: u, Name: n} }
func cat(name string, es ...*Ent) *Cat    { return &Cat{Name: name, Entries: es} }
func col(cs ...*Cat) *Col                 { return &Col{Categories: cs} }
func colsBlock(cs ...*Col) *Blk           { return &Blk{Columns: cs} }
func hrBlock() *Blk                       { return &Blk{HR: true} }
func page(bs ...*Blk) *Pg                 { return &Pg{Blocks: bs} }
func tabPage(name string, bs ...*Blk) *Pg { return &Pg{Tab: name, Blocks: bs} }

func Test_preprocessBookmarks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []*Pg
	}{
		{
			name:  "basic",
			input: "Category: Search\nhttp://g.com G\nCategory: Wikies\nhttp://w.com W\n",
			want: []*Pg{
				page(colsBlock(
					col(cat("Search", e("http://g.com", "G")),
						cat("Wikies", e("http://w.com", "W"))),
				)),
			},
		},
		{
			name:  "columns",
			input: "Category: Search\nhttp://g.com G\nColumn\nCategory: Wikies\nhttp://w.com W\n",
			want: []*Pg{
				page(colsBlock(
					col(cat("Search", e("http://g.com", "G"))),
					col(cat("Wikies", e("http://w.com", "W"))),
				)),
			},
		},
		{
			name:  "pages",
			input: "Category: A\nhttp://a.com a\nPage\nCategory: B\nhttp://b.com b\n",
			want: []*Pg{
				page(colsBlock(col(cat("A", e("http://a.com", "a"))))),
				page(colsBlock(col(cat("B", e("http://b.com", "b"))))),
			},
		},
		{
			name:  "pages and columns",
			input: "Category: A\nhttp://a.com\nColumn\nCategory: B\nhttp://b.com\nPage\nCategory: C\nhttp://c.com\nColumn\nCategory: D\nhttp://d.com\n",
			want: []*Pg{
				page(colsBlock(
					col(cat("A", e("http://a.com", "http://a.com"))),
					col(cat("B", e("http://b.com", "http://b.com"))),
				)),
				page(colsBlock(
					col(cat("C", e("http://c.com", "http://c.com"))),
					col(cat("D", e("http://d.com", "http://d.com"))),
				)),
			},
		},
		{
			name:  "horizontal rule",
			input: "Category: One\nhttp://one.com\n--\nCategory: Two\nhttp://two.com\n",
			want: []*Pg{
				page(
					colsBlock(col(cat("One", e("http://one.com", "http://one.com")))),
					hrBlock(),
					colsBlock(col(cat("Two", e("http://two.com", "http://two.com")))),
				),
			},
		},
		{
			name:  "tabs",
			input: "Tab: First\nCategory: A\nTab: Second\nCategory: B\n",
			want: []*Pg{
				page(colsBlock(col())),
				tabPage("First", colsBlock(col(cat("A")))),
				tabPage("Second", colsBlock(col(cat("B")))),
			},
		},
		{
			name:  "tab multiple pages",
			input: "Tab: X\nCategory: A\nPage\nCategory: B\n",
			want: []*Pg{
				page(colsBlock(col())),
				tabPage("X", colsBlock(col(cat("A")))),
				tabPage("X", colsBlock(col(cat("B")))),
			},
		},
		{
			name:  "anonymous tab",
			input: "Tab: F\nCategory: A\nTab\nCategory: B\n",
			want: []*Pg{
				page(colsBlock(col())),
				tabPage("F", colsBlock(col(cat("A")))),
				page(colsBlock(col(cat("B")))),
			},
		},
		{
			name:  "tab no colon with name",
			input: "Tab Foo\nCategory: A\n",
			want: []*Pg{
				page(colsBlock(col())),
				tabPage("Foo", colsBlock(col(cat("A")))),
			},
		},
		{
			name:  "page name no colon",
			input: "Page Start\nCategory: A\nPage End\nCategory: B\n",
			want: []*Pg{
				page(colsBlock(col())),
				page(colsBlock(col(cat("A")))),
				page(colsBlock(col(cat("B")))),
			},
		},
		{
			name:  "anonymous categories",
			input: "Category:\nhttp://a.com\nCategory:\nhttp://b.com\n",
			want: []*Pg{
				page(colsBlock(col(
					cat("", e("http://a.com", "http://a.com")),
					cat("", e("http://b.com", "http://b.com")),
				))),
			},
		},
	}

	ignore := cmpopts.IgnoreFields(BookmarkCategory{}, "Index")
	ignorePage := cmpopts.IgnoreFields(BookmarkPage{}, "Tab", "Name")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessBookmarks(tt.input)
			if diff := cmp.Diff(tt.want, got, ignore, ignorePage); diff != "" {
				t.Errorf("diff:\n%s", diff)
			}
		})
	}
}

func Test_preprocessBookmarksIndices(t *testing.T) {
	input := "Category: A\nColumn\nCategory: B\nPage\nCategory: C\n"
	pages := PreprocessBookmarks(input)
	var got []int
	for _, p := range pages {
		for _, b := range p.Blocks {
			if b.HR {
				continue
			}
			for _, c := range b.Columns {
				for _, cat := range c.Categories {
					got = append(got, cat.Index)
				}
			}
		}
	}
	expected := []int{0, 1, 2}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("diff:\n%s", diff)
	}
}

func Test_preprocessBookmarksPageNames(t *testing.T) {
	input := "Page: Start\nCategory: A\nPage: End\nCategory: B\n"
	pages := PreprocessBookmarks(input)
	if len(pages) < 3 {
		t.Fatalf("expected 3 pages got %d", len(pages))
	}
	if pages[1].Name != "Start" {
		t.Fatalf("expected Start got %q", pages[1].Name)
	}
	if pages[2].Name != "End" {
		t.Fatalf("expected End got %q", pages[2].Name)
	}
}
