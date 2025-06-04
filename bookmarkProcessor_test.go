package gobookmarks

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func Test_preprocessBookmarks(t *testing.T) {
	tests := []struct {
		name      string
		bookmarks string
		want      []*BookmarkPage
	}{
		{
			name:      "basic",
			bookmarks: "Category: Search\nhttp://www.google.com.au Google\nCategory: Wikies\nhttp://en.wikipedia.org/wiki/Main_Page Wikipedia\nhttp://mathworld.wolfram.com/ Math World\nhttp://gentoo-wiki.com/Main_Page Gentoo-wiki\n",
			want: []*BookmarkPage{{
				Columns: []*BookmarkColumn{{
					Categories: []*BookmarkCategory{
						{
							Name:    "Search",
							Entries: []*BookmarkEntry{{Url: "http://www.google.com.au", Name: "Google"}},
						},
						{
							Name: "Wikies",
							Entries: []*BookmarkEntry{
								{Url: "http://en.wikipedia.org/wiki/Main_Page", Name: "Wikipedia"},
								{Url: "http://mathworld.wolfram.com/", Name: "Math World"},
								{Url: "http://gentoo-wiki.com/Main_Page", Name: "Gentoo-wiki"},
							},
						},
					},
				}},
			}},
		},
		{
			name:      "columns",
			bookmarks: "Category: Search\nhttp://www.google.com.au Google\nColumn\nCategory: Wikies\nhttp://en.wikipedia.org/wiki/Main_Page Wikipedia\n",
			want: []*BookmarkPage{{
				Columns: []*BookmarkColumn{
					{
						Categories: []*BookmarkCategory{
							{
								Name:    "Search",
								Entries: []*BookmarkEntry{{Url: "http://www.google.com.au", Name: "Google"}},
							},
						},
					},
					{
						Categories: []*BookmarkCategory{
							{
								Name:    "Wikies",
								Entries: []*BookmarkEntry{{Url: "http://en.wikipedia.org/wiki/Main_Page", Name: "Wikipedia"}},
							},
						},
					},
				},
			}},
		},
		{
			name:      "pages",
			bookmarks: "Category: First\nhttp://example.com A\nPage\nCategory: Second\nhttp://example.org B\n",
			want: []*BookmarkPage{
				{
					Columns: []*BookmarkColumn{{
						Categories: []*BookmarkCategory{
							{
								Name:    "First",
								Entries: []*BookmarkEntry{{Url: "http://example.com", Name: "A"}},
							},
						},
					}},
				},
				{
					Columns: []*BookmarkColumn{{
						Categories: []*BookmarkCategory{
							{
								Name:    "Second",
								Entries: []*BookmarkEntry{{Url: "http://example.org", Name: "B"}},
							},
						},
					}},
				},
			},
		},
		{
			name:      "pages and columns",
			bookmarks: "Category: A\nhttp://a.com\nColumn\nCategory: B\nhttp://b.com\nPage\nCategory: C\nhttp://c.com\nColumn\nCategory: D\nhttp://d.com\n",
			want: []*BookmarkPage{
				{
					Columns: []*BookmarkColumn{
						{
							Categories: []*BookmarkCategory{
								{Name: "A", Entries: []*BookmarkEntry{{Url: "http://a.com", Name: "http://a.com"}}},
							},
						},
						{
							Categories: []*BookmarkCategory{
								{Name: "B", Entries: []*BookmarkEntry{{Url: "http://b.com", Name: "http://b.com"}}},
							},
						},
					},
				},
				{
					Columns: []*BookmarkColumn{
						{
							Categories: []*BookmarkCategory{
								{Name: "C", Entries: []*BookmarkEntry{{Url: "http://c.com", Name: "http://c.com"}}},
							},
						},
						{
							Categories: []*BookmarkCategory{
								{Name: "D", Entries: []*BookmarkEntry{{Url: "http://d.com", Name: "http://d.com"}}},
							},
						},
					},
				},
			},
		},
		{
			name:      "double dash compat",
			bookmarks: "Category: One\nhttp://one.com\n--\nCategory: Two\nhttp://two.com\n",
			want: []*BookmarkPage{
				{
					Columns: []*BookmarkColumn{{
						Categories: []*BookmarkCategory{{Name: "One", Entries: []*BookmarkEntry{{Url: "http://one.com", Name: "http://one.com"}}}},
					}},
				},
				{
					Columns: []*BookmarkColumn{{
						Categories: []*BookmarkCategory{{Name: "Two", Entries: []*BookmarkEntry{{Url: "http://two.com", Name: "http://two.com"}}}},
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessBookmarks(tt.bookmarks)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("PreprocessBookmarks() = diff\n%s", diff)
			}
		})
	}
}
