package gobookmarks_test

import (
	"github.com/arran4/gobookmarks"
	"strings"
	"testing"
)

func TestParseParityComments(t *testing.T) {
	input := "## File preamble\nTab: Work ## Tab comment\nPage: Main ## Page comment\nColumn: Daily ## Column comment\nCategory: Development ## Category comment\nhttps://example.com/path\\##literal Example \\## literal ## Entry comment\n## Between entries\nhttps://github.com GitHub\n## Before next category\nCategory: Reference\nhttps://example.org Reference\n-- ## Separator comment\n## After separator\nColumn: Archive\nCategory: Old\nhttps://old.example Old\n## EOF comment"

	bl1 := gobookmarks.ParseBookmarks(input)
	bl2, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}

	if strings.TrimSpace(bl1.String()) != strings.TrimSpace(bl2.String()) {
		t.Fatalf("parsers are not equivalent:\nParse:\n%v\n\nStrictParse:\n%v", bl1.String(), bl2.String())
	}

	// Verify JSON mapping
	j1 := bl1.ToJSON()

	bl3, err2 := gobookmarks.BookmarkListFromJSON(j1)
	if err2 != nil {
		t.Fatalf("JSON decode failed: %v\n\nOriginal:\n%v", err2, bl1.String())
	}
	if strings.TrimSpace(bl3.String()) != strings.TrimSpace(bl1.String()) {
		t.Fatalf("JSON roundtrip failed:\nOriginal:\n%v\n\nRoundtrip:\n%v", bl1.String(), bl3.String())
	}

	// Verify escaping decoded properly
	e1 := bl1[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]
	if e1.Url != "https://example.com/path\\##literal" {
		t.Errorf("Url escaping wrong: %v", e1.Url)
	}
	if e1.Name != "Example ## literal" {
		t.Errorf("Name escaping wrong: %v", e1.Name)
	}
	if !e1.HasInlineComment || e1.InlineComment != " Entry comment" {
		t.Errorf("Inline comment wrong: %v", e1.InlineComment)
	}
}

func TestParseEditingComments(t *testing.T) {
	input := "## File preamble\nTab: Work ## Tab comment\nPage: Main ## Page comment\nColumn: Daily ## Column comment\nCategory: Development ## Category comment\nhttps://example.com/path\\##literal Example \\## literal ## Entry comment\n## Between entries\nhttps://github.com GitHub\n## Before next category\nCategory: Reference\nhttps://example.org Reference\n-- ## Separator comment\n## After separator\nColumn: Archive\nCategory: Old\nhttps://old.example Old\n## EOF comment"

	bl := gobookmarks.ParseBookmarks(input)

	// Extracting a page and saving it unchanged
	pageText, pageName, err := gobookmarks.ExtractPage(input, 0, 0)
	if err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}
	if strings.Contains(pageText, "## File preamble") {
		t.Errorf("ExtractPage should not include page leading comments: %v", pageText)
	}

	pageParsed := gobookmarks.ParseBookmarks("Tab\nPage: " + pageName + "\n" + pageText)
	p := pageParsed[0].Pages[0]
	// Emulate PageEditSaveAction logic manually:
	origPage := bl[0].Pages[0]
	p.LeadingComments = origPage.LeadingComments
	p.InlineComment = origPage.InlineComment
	p.HasInlineComment = origPage.HasInlineComment
	bl[0].Pages[0] = p

	if strings.TrimSpace(bl.String()) != strings.TrimSpace(input) {
		t.Errorf("Extract+Save unchanged failed:\nExpected:\n%v\nGot:\n%v", input, bl.String())
	}

	// Replace a category without altering adjacent comments
	catText, err := gobookmarks.ExtractCategoryByIndex(input, 1) // Reference
	if err != nil {
		t.Fatalf("ExtractCategoryByIndex: %v", err)
	}
	newCatText := strings.ReplaceAll(catText, "Reference", "Replaced")
	newInput, err := gobookmarks.ReplaceCategoryByIndex(input, 1, newCatText)
	if err != nil {
		t.Fatalf("ReplaceCategoryByIndex: %v", err)
	}

	if !strings.Contains(newInput, "Category: Replaced") {
		t.Errorf("Replace failed: %v", newInput)
	}
	if !strings.Contains(newInput, "## Before next category") {
		t.Errorf("Replace dropped leading comment: %v", newInput)
	}
	if !strings.Contains(newInput, "-- ## Separator comment") {
		t.Errorf("Replace damaged following separator: %v", newInput)
	}
}

func TestParseCommentsEmpty(t *testing.T) {
	input := "## First\n##\n## Last"

	bl1 := gobookmarks.ParseBookmarks(input)
	bl2, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}

	if strings.TrimSpace(bl1.String()) != strings.TrimSpace(bl2.String()) {
		t.Fatalf("parsers are not equivalent:\nParse:\n%v\n\nStrictParse:\n%v", bl1.String(), bl2.String())
	}

	// Verify JSON mapping
	j1 := bl1.ToJSON()

	bl3, err2 := gobookmarks.BookmarkListFromJSON(j1)
	if err2 != nil {
		t.Fatalf("JSON decode failed: %v\n\nOriginal:\n%v", err2, bl1.String())
	}
	if strings.TrimSpace(bl3.String()) != strings.TrimSpace(bl1.String()) {
		t.Fatalf("JSON roundtrip failed:\nOriginal:\n%v\n\nRoundtrip:\n%v", bl1.String(), bl3.String())
	}
}
