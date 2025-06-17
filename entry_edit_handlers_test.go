package gobookmarks

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEditEntryPageTrailingSlashRef(t *testing.T) {
	_, _, ctx := setupHandlerTest(t, "Category: A\nhttps://a A\n")
	req := httptest.NewRequest("GET", "/editEntry?cat=0&entry=0&ref=refs/heads/main/", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := EditEntryPage(w, req); err != nil {
		t.Fatalf("EditEntryPage: %v", err)
	}
	if w.Code != 200 {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Name") {
		t.Fatalf("output missing: %s", w.Body.String())
	}
	// ensure provider works with trailing slash by fetching bookmarks again
	if _, _, err := GetBookmarks(ctx, "alice", "refs/heads/main/", nil); err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
}
