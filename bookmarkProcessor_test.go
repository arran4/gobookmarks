package main

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func Test_preprocessBookmarks(t *testing.T) {
	tests := []struct {
		name      string
		bookmarks string
		want      []*BookmarkColumn
	}{
		{
			name:      "Test",
			bookmarks: "Category: Search\nhttp://www.google.com.au Google\nCategory: Wikies\nhttp://en.wikipedia.org/wiki/Main_Page Wikipedia\nhttp://mathworld.wolfram.com/ Math World\nhttp://gentoo-wiki.com/Main_Page Gentoo-wiki\n",
			want: []*BookmarkColumn{{
				Categories: []*BookmarkCategory{
					{
						Name: "Search",
						Entries: []*BookmarkEntry{
							{
								Url:  "http://www.google.com.au",
								Name: "Google",
							},
						},
					},
					{
						Name: "Wikies",
						Entries: []*BookmarkEntry{
							{
								Url:  "http://en.wikipedia.org/wiki/Main_Page",
								Name: "Wikipedia",
							},
							{
								Url:  "http://mathworld.wolfram.com/",
								Name: "Math World",
							},
							{
								Url:  "http://gentoo-wiki.com/Main_Page",
								Name: "Gentoo-wiki",
							},
						},
					},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessBookmarks(tt.bookmarks)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("preprocessBookmarks() = diff\n%s", diff)
			}
		})
	}
}
