package gobookmarks

import (
	"strings"
	"testing"
)

func TestStrictParseBookmarks(t *testing.T) {
	// Valid inputs
	validInputs := []string{
		"Category: Search\nhttps://google.com Google",
		"Tab: A\nPage: B\nCategory: C\nsearch:foo foo",
		"Category: X\n/local/path Local Path",
		"Tab\nPage\n--\nColumn\nCategory: Cat\nhttp://link",
		"Category: Unnamed\nhttp://link",
		"Category: Test\nexample.com",       // Single token should be parsed as a link (even without scheme)
		"Category: Test\nftp://test",        // Other schemes
		"Category: Valid\npage.example.com", // Entry token starting with directive prefix but valid
		"Category: Valid\ncolumnist.example",
		"Category: Valid\ntabby",
		"Category: Valid\ncategory.example",
		"http://link", // should fail if no category, but wait. If no category, StrictParse rejects it. Wait, ParseBookmarks silently drops it. StrictParse now rejects it!
	}

	// Test the final valid one to ensure it fails
	_, err := StrictParseBookmarks(validInputs[len(validInputs)-1])
	if err == nil {
		t.Errorf("Expected link outside category to fail")
	}

	for _, input := range validInputs[:len(validInputs)-1] {
		_, err := StrictParseBookmarks(input)
		if err != nil {
			t.Errorf("Expected valid input to parse successfully, got: %v\nInput: %q", err, input)
		}
	}

	invalidInputs := []string{
		"UnknownDirective",
		"Tab: My Tab\nUnknownDirective",
		"Category: Valid\nPagge: Missing", // Misspelled directive inside a category
		"Category: Valid\nCategor: Missing",
		"Category: Valid\nColum:",
	}

	for _, input := range invalidInputs {
		_, err := StrictParseBookmarks(input)
		if err == nil {
			t.Errorf("Expected invalid input to fail parsing, got success\nInput: %q", input)
		} else if !strings.Contains(err.Error(), "unrecognized directive") && !strings.Contains(err.Error(), "malformed link") && !strings.Contains(err.Error(), "outside of category") && !strings.Contains(err.Error(), "malformed") && !strings.Contains(err.Error(), "misspelled") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	}
}
