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
	entry := &gobookmarks.JSONEntry{} // zero values
	cat := &gobookmarks.JSONCategory{}
	blk := &gobookmarks.JSONBlock{}

	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "\"url\":\"\"") {
		t.Fatalf("Missing zero-value required field 'url' in JSONEntry: %s", s)
	}
	if !strings.Contains(s, "\"name\":\"\"") {
		t.Fatalf("Missing zero-value required field 'name' in JSONEntry: %s", s)
	}

	b, _ = json.Marshal(cat)
	s = string(b)
	if !strings.Contains(s, "\"name\":\"\"") {
		t.Fatalf("Missing zero-value required field 'name' in JSONCategory: %s", s)
	}

	b, _ = json.Marshal(blk)
	s = string(b)
	if !strings.Contains(s, "\"hr\":false") {
		t.Fatalf("Missing zero-value required field 'hr' in JSONBlock: %s", s)
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
	if entry.LeadingComments[0] != ` comment with \## inside` {
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

func TestPermissiveStrictParity(t *testing.T) {
	input := `## leading tab
tab: Tab1 ## inline tab
## leading hr
-- ## inline hr
## leading col
Column: Column 1 ## inline col
## leading cat
Category: Category 1 ## inline cat
## leading entry
https://example.com Example 1 ## inline entry
`
	strictList, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Strict parse failed: %v", err)
	}

	permissiveList := gobookmarks.ParseBookmarks(input)

	strictJSON, _ := json.MarshalIndent(strictList.ToJSON(), "", "  ")
	permissiveJSON, _ := json.MarshalIndent(permissiveList.ToJSON(), "", "  ")

	if string(strictJSON) != string(permissiveJSON) {
		t.Fatalf("JSON parity failed.\nStrict:\n%s\nPermissive:\n%s", strictJSON, permissiveJSON)
	}
}

func TestEmptyInlineComments(t *testing.T) {
	input := `tab: Tab1 ##
-- ##
Column: Column 1 ##
Category: Category 1 ##
https://example.com Example 1 ##
`

	list, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tab := list[0]
	if !tab.HasInlineComment || strings.TrimSpace(tab.InlineComment) != "" {
		t.Fatalf("Tab empty inline comment failed")
	}

	page := tab.Pages[0]
	hrBlock := page.Blocks[1]
	if !hrBlock.HasInlineComment || strings.TrimSpace(hrBlock.InlineComment) != "" {
		t.Fatalf("HR empty inline comment failed")
	}

	col := page.Blocks[2].Columns[0]
	if !col.HasInlineComment || strings.TrimSpace(col.InlineComment) != "" {
		t.Fatalf("Column empty inline comment failed")
	}

	cat := col.Categories[0]
	if !cat.HasInlineComment || strings.TrimSpace(cat.InlineComment) != "" {
		t.Fatalf("Category empty inline comment failed")
	}

	entry := cat.Entries[0]
	if !entry.HasInlineComment || strings.TrimSpace(entry.InlineComment) != "" {
		t.Fatalf("Entry empty inline comment failed")
	}
}

func TestCategoryEditPreservation(t *testing.T) {
	// A no-op ExtractCategoryByIndex -> ReplaceCategoryByIndex should reproduce the exact original bytes
	input := `tab: Tab1

## leading cat
Category: Category 1

https://example.com/1
https://example.com/2


`

	catStr, _ := gobookmarks.ExtractCategoryByIndex(input, 0)

	// Ensure that string replacement logic correctly preserves everything
	replaced, err := gobookmarks.ReplaceCategoryByIndex(input, 0, catStr)
	if err != nil {
		t.Fatalf("Replace failed: %v", err)
	}

	if replaced != input {
		t.Fatalf("Category replacement modified original formatting.\nExpected:\n%q\nGot:\n%q", input, replaced)
	}
}

func TestBackslashEscaping(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string // the semantic value we want after parsing
	}{
		{"0 slashes", "tab: tab\nCategory\nhttps://example.com \\##", `##`},
		{"1 slash", "tab: tab\nCategory\nhttps://example.com \\\\##", `\##`},
		{"2 slashes", "tab: tab\nCategory\nhttps://example.com \\\\\\##", `\\##`},
		{"3 slashes", "tab: tab\nCategory\nhttps://example.com \\\\\\\\##", `\\\##`},
		{"embedded url", "tab: tab\nCategory\nhttps://example.com/##hash ## comment", `https://example.com/##hash`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list, err := gobookmarks.StrictParseBookmarks(tc.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			entry := list[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]

			val := entry.Name
			if tc.name == "embedded url" {
				val = entry.Url
			}

			if val != tc.expected {
				t.Fatalf("Expected semantic value %q, got %q", tc.expected, val)
			}

			// Test round trip
			roundtripStr := list.String()
			list2, err := gobookmarks.StrictParseBookmarks(roundtripStr)
			if err != nil {
				t.Fatalf("Roundtrip parse failed: %v", err)
			}

			entry2 := list2[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]
			val2 := entry2.Name
			if tc.name == "embedded url" {
				val2 = entry2.Url
			}

			if val2 != tc.expected {
				t.Fatalf("Roundtrip failed. Expected semantic value %q, got %q", tc.expected, val2)
			}

			// JSON round trip
			jsonBytes, _ := json.Marshal(list.ToJSON())
			var jsonList []*gobookmarks.JSONTab
			json.Unmarshal(jsonBytes, &jsonList)
			nativeList, _ := gobookmarks.BookmarkListFromJSON(jsonList)

			entry3 := nativeList[0].Pages[0].Blocks[0].Columns[0].Categories[0].Entries[0]
			val3 := entry3.Name
			if tc.name == "embedded url" {
				val3 = entry3.Url
			}
			if val3 != tc.expected {
				t.Fatalf("JSON roundtrip failed. Expected semantic value %q, got %q", tc.expected, val3)
			}
		})
	}
}

func TestPageEmptyInlineComments(t *testing.T) {
	input := `tab: Tab1
page: Page 1 ## page comment
`
	list, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	page := list[0].Pages[0]
	if !page.HasInlineComment || strings.TrimSpace(page.InlineComment) != "page comment" {
		t.Fatalf("Page empty inline comment failed")
	}
}

func TestPageEmptyInlineCommentsMatrix(t *testing.T) {
	input := `tab: Tab1
page: Page 1 ##
`
	list, err := gobookmarks.StrictParseBookmarks(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	page := list[0].Pages[0]
	if !page.HasInlineComment || strings.TrimSpace(page.InlineComment) != "" {
		t.Fatalf("Page empty inline comment failed")
	}
}
