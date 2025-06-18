package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCategoryMoveAction(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", shaComplex); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	text, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	tabs := ParseBookmarks(text)
	pageSha := tabs[0].Pages[0].Sha()
	form := url.Values{"from": {"0"}, "to": {"1"}, "branch": {"main"}, "ref": {"refs/heads/main"}, "pageSha": {pageSha}}
	req := httptest.NewRequest("POST", "/moveCategory", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryMoveAction(w, req); err != nil {
		t.Fatalf("CategoryMoveAction: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	tabs = ParseBookmarks(shaComplex)
	if err := tabs.MoveCategory(0, 1, false); err != nil {
		t.Fatalf("MoveCategory local: %v", err)
	}
	expected := tabs.String()
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
	if sha == "" {
		t.Log(sha)
	}
}

func TestCategoryMoveActionConcurrent(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", shaComplex); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	text, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	tabs := ParseBookmarks(text)
	pageSha := tabs[0].Pages[0].Sha()
	// modify first page so sha changes
	tabs[0].Pages[0].Blocks[0].Columns[0].Categories[0].Name = "X"
	modified := tabs.String()
	if err := p.UpdateBookmarks(context.Background(), user, nil, "refs/heads/main", "main", modified, sha); err != nil {
		t.Fatalf("UpdateBookmarks: %v", err)
	}

	form := url.Values{"from": {"0"}, "to": {"1"}, "branch": {"main"}, "ref": {"refs/heads/main"}, "pageSha": {pageSha}}
	req := httptest.NewRequest("POST", "/moveCategory", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	err = CategoryMoveAction(w, req)
	if err == nil || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("expected concurrency error, got %v", err)
	}
}
