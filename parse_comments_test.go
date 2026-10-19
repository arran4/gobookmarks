package gobookmarks_test

import (
	"github.com/arran4/gobookmarks"
	"testing"
)

func TestStrictParseComments(t *testing.T) {
	input := `## Tab lead
Tab: tab1 ## Tab inline
## Page lead
Page: page1 ## Page inline
## Block lead
-- ## Block inline
## Col lead
Column: col1 ## Col inline
## Cat lead
Category: cat1 ## Cat inline
## Entry lead
http://example.com/1 Link 1 ## Entry inline
## Trailing comment`

	bl, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}

	if len(bl) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(bl))
	}
	tab := bl[0]
	if len(tab.LeadingComments) != 1 || tab.LeadingComments[0] != " Tab lead" {
		t.Errorf("tab leading comments wrong: %v", tab.LeadingComments)
	}
	if tab.InlineComment != " Tab inline" {
		t.Errorf("tab inline comment wrong: %q", tab.InlineComment)
	}

	if len(tab.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tab.Pages))
	}
	page := tab.Pages[0]
	if len(page.LeadingComments) != 1 || page.LeadingComments[0] != " Page lead" {
		t.Errorf("page leading comments wrong: %v", page.LeadingComments)
	}
	if page.InlineComment != " Page inline" {
		t.Errorf("page inline comment wrong: %q", page.InlineComment)
	}

	if len(page.Blocks) != 3 {
		t.Fatalf("expected 2 blocks, got %d", len(page.Blocks))
	}
	blk := page.Blocks[1]
	if len(blk.LeadingComments) != 1 || blk.LeadingComments[0] != " Block lead" {
		t.Errorf("block leading comments wrong: %v", blk.LeadingComments)
	}
	if blk.InlineComment != " Block inline" {
		t.Errorf("block inline comment wrong: %q", blk.InlineComment)
	}

	blk2 := page.Blocks[2]
	col := blk2.Columns[0]
	if len(col.LeadingComments) != 1 || col.LeadingComments[0] != " Col lead" {
		t.Errorf("col leading comments wrong: %v", col.LeadingComments)
	}
	if col.InlineComment != " Col inline" {
		t.Errorf("col inline comment wrong: %q", col.InlineComment)
	}

	if len(col.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(col.Categories))
	}
	cat := col.Categories[0]
	if len(cat.LeadingComments) != 1 || cat.LeadingComments[0] != " Cat lead" {
		t.Errorf("cat leading comments wrong: %v", cat.LeadingComments)
	}
	if cat.InlineComment != " Cat inline" {
		t.Errorf("cat inline comment wrong: %q", cat.InlineComment)
	}

	if len(cat.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cat.Entries))
	}
	entry := cat.Entries[0]
	if len(entry.LeadingComments) != 1 || entry.LeadingComments[0] != " Entry lead" {
		t.Errorf("entry leading comments wrong: %v", entry.LeadingComments)
	}
	if entry.InlineComment != " Entry inline" {
		t.Errorf("entry inline comment wrong: %q", entry.InlineComment)
	}

	if len(tab.TrailingComments) != 1 || tab.TrailingComments[0] != " Trailing comment" {
		t.Errorf("trailing comments wrong: %v", tab.TrailingComments)
	}
}

func TestStrictParseInvalidComments(t *testing.T) {
	input := `## Valid comment
Invalid Text
Category: valid`

	_, err := gobookmarks.StrictParseBookmarks(input)
	if err == nil {
		t.Fatal("expected strict parser to fail on invalid non-comment content")
	}
}

func TestStrictParseEmptyCommentDoc(t *testing.T) {
	input := `## Comment only`
	bl, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("strict parse empty comment doc failed: %v", err)
	}
	if len(bl) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(bl))
	}
	tab := bl[0]
	if len(tab.LeadingComments) != 1 || tab.LeadingComments[0] != " Comment only" {
		t.Errorf("wrong comments: %v", tab.LeadingComments)
	}
}
