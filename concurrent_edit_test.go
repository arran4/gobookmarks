package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBookmarksEditSaveActionConcurrent(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)

	original := "Category: A\nhttp://one.com one"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha1, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks sha1: %v", err)
	}

	updated := "Category: B\nhttp://two.com two"
	if err := p.UpdateBookmarks(context.Background(), user, nil, "refs/heads/main", "main", updated, sha1); err != nil {
		t.Fatalf("UpdateBookmarks: %v", err)
	}
	_, sha2, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks sha2: %v", err)
	}
	if sha1 == sha2 {
		t.Fatalf("SHA did not change")
	}

	form := url.Values{"text": {"Category: C\nhttp://three.com three"}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha1}}
	req := httptest.NewRequest("POST", "/edit/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	err = BookmarksEditSaveAction(w, req)
	if err == nil || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("expected concurrency error, got %v", err)
	}

	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks final: %v", err)
	}
	if got != updated {
		t.Fatalf("bookmarks changed unexpectedly: %q", got)
	}
}
