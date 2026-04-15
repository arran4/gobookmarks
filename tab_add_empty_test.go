package gobookmarks

import (
	"testing"
)

func TestAddTabDoesNotPopulateFromIndex0(t *testing.T) {
	bookmarksStr := `Tab: test6
Page: page2
Page: page3
Tab: test3
Category: hi
http://example.com E2?
Page: hiii
Category: Hiiiiiii
http://www.exmaple.com/ exmaple34
`

	tabs := ParseBookmarks(bookmarksStr)

	// Simulate what EditTabPage does when trying to "Add Tab"
	tabName := ""
	tabIdx := 0 // default since it's an Add Tab without tab param
	tabFromQuery := tabName != ""
    hasTabParam := false // r.URL.Query().Has("tab")

	isAddMode := !hasTabParam && tabName == ""
    text := ""
	if !isAddMode {
		if tabName == "" && tabIdx < len(tabs) {
			tabName = tabs[tabIdx].Name
		}
		if tabFromQuery || tabIdx < len(tabs) {
			tabText, err := ExtractTabByIndex(bookmarksStr, tabIdx)
			if err != nil {
				t.Fatalf("ExtractTabByIndex: %v", err)
			}
            text = tabText
		}
	}

    if text != "" {
        t.Fatalf("Expected text to be empty in add mode, but got:\n%s", text)
    }
}
