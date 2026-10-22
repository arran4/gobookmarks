package gobookmarks_test

import (
	"encoding/json"
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

func TestStringEscaping(t *testing.T) {
	// P0: native -> String -> native equality for specific case
	entry := &gobookmarks.BookmarkEntry{
		Url:             "https://example.com",
		Name:            "Example",
		LeadingComments: []string{"first ## second"},
	}

	serialized := entry.String()
	expectedString := "##first ## second\nhttps://example.com Example\n"
	if serialized != expectedString {
		t.Fatalf("Expected %q, got %q", expectedString, serialized)
	}

	fullString := "tab:\nCategory\n" + serialized
	parsed, err := gobookmarks.StrictParseBookmarks(fullString)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(parsed) != 1 || len(parsed[0].Pages) != 1 || len(parsed[0].Pages[0].Blocks) != 1 || len(parsed[0].Pages[0].Blocks[0].Columns) != 1 || len(parsed[0].Pages[0].Blocks[0].Columns[0].Categories) != 1 || len(parsed[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries) != 1 {
		t.Fatalf("Unexpected parsed structure")
	}

	parsedEntry := parsed[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]
	if len(parsedEntry.LeadingComments) != 1 || parsedEntry.LeadingComments[0] != "first ## second" {
		t.Fatalf("Expected leading comment 'first ## second', got %v", parsedEntry.LeadingComments)
	}
}

func TestJSONCompatibility(t *testing.T) {
	// P1: JSON wire compatibility - old fields should be present without omitempty
	entry := &gobookmarks.JSONEntry{
		URL:  "http://example.com",
		Name: "Example",
	}

	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	s := string(b)
	if !strings.Contains(s, "\"url\":\"http://example.com\"") {
		t.Fatalf("Missing required field 'url' in JSON: %s", s)
	}
	if !strings.Contains(s, "\"name\":\"Example\"") {
		t.Fatalf("Missing required field 'name' in JSON: %s", s)
	}
}

func TestLiteralHashHash(t *testing.T) {
	input := `tab:
Category
## comment with \## inside
https://example.com/##hash ## inline comment with \## inside
`
	parsed, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	entry := parsed[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]
	if entry.Url != "https://example.com/##hash" {
		t.Fatalf("URL was corrupted: %q", entry.Url)
	}
	if entry.LeadingComments[0] != ` comment with ## inside` {
		t.Fatalf("Leading comment was corrupted: %q", entry.LeadingComments[0])
	}
	if entry.InlineComment != ` inline comment with \## inside` {
		t.Fatalf("Inline comment was corrupted: %q", entry.InlineComment)
	}
}

func TestNativeJSONRoundtrip(t *testing.T) {
	input := `## leading tab
Tab: Tab1 ## inline tab
## leading hr
-- ## inline hr
## leading col
Column: Column 1 ## inline col
## leading cat
Category: Category 1 ## inline cat
## leading entry
https://example.com Example 1 ## inline entry
`
	parsed, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	jsonBytes, err := json.Marshal(parsed.ToJSON())
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var jsonList []*gobookmarks.JSONTab
	if err := json.Unmarshal(jsonBytes, &jsonList); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	nativeList, err := gobookmarks.BookmarkListFromJSON(jsonList)
	if err != nil {
		t.Fatalf("Failed to convert JSON to Native: %v", err)
	}

	outString := nativeList.String()
	if strings.TrimSpace(outString) != strings.TrimSpace(input) {
		t.Fatalf("JSON roundtrip lost formatting. Expected:\n%s\nGot:\n%s", input, outString)
	}
}
