package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCategoryAddSaveAction(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	original := "Category: First\nhttp://one.com one\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}

	form := url.Values{"text": {"Category: New\nhttp://two.com two"}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}, "tab": {"0"}, "page": {"0"}, "col": {"0"}}
	req := httptest.NewRequest("POST", "/addCategory", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryAddSaveAction(w, req); err != nil {
		t.Fatalf("CategoryAddSaveAction: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	expected := original + "Category: New\nhttp://two.com two\n"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}
