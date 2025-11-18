package gobookmarks

import "fmt"

// ExtractPage returns the text and name for a page located at tabIdx/pageIdx.
func ExtractPage(bookmarks string, tabIdx, pageIdx int) (string, string, error) {
	tabs := ParseBookmarks(bookmarks)
	if tabIdx < 0 || tabIdx >= len(tabs) {
		return "", "", fmt.Errorf("tab index %d out of range", tabIdx)
	}
	pages := tabs[tabIdx].Pages
	if pageIdx < 0 || pageIdx >= len(pages) {
		return "", "", fmt.Errorf("page index %d out of range", pageIdx)
	}
	page := pages[pageIdx]
	return page.String(), page.Name, nil
}
