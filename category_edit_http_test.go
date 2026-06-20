package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func setupCategoryEditTest(t *testing.T) (GitProvider, string, *sessions.Session, context.Context) {
	tmp := t.TempDir()
	Config.LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	if err := p.CreateRepo(context.Background(), user, nil, Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	Config.SessionName = "testsession"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	sessReq := httptest.NewRequest("GET", "/", nil)
	sess, err := getSession(httptest.NewRecorder(), sessReq)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	sess.Values["GithubUser"] = &User{Login: user}
	sess.Values["Token"] = &oauth2.Token{}
	ctx := context.WithValue(sessReq.Context(), ContextValues("session"), sess)
	ctx = context.WithValue(ctx, ContextValues("provider"), "git")
	ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{})
	return p, user, sess, ctx
}

func TestCategoryEditSaveAction(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	original := "Category: First\nhttp://one.com one\nCategory: Second\nhttp://two.com two\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}

	newSection := "Category: Second\nhttp://changed.com x"
	form := url.Values{"text": {newSection}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}}
	req := httptest.NewRequest("POST", "/editCategory?index=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryEditSaveAction(w, req); err != nil {
		t.Fatalf("CategoryEditSaveAction: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	expected := "Category: First\nhttp://one.com one\n" + newSection
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestCategoryEditSaveActionAnonymous(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	original := "Category:\nhttp://one.com\nCategory: Named\nhttp://two.com\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	newSection := "Category:\nhttp://changed.com"
	form := url.Values{"text": {newSection}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}}
	req := httptest.NewRequest("POST", "/editCategory?index=0", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryEditSaveAction(w, req); err != nil {
		t.Fatalf("CategoryEditSaveAction: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	expected := newSection + "\nCategory: Named\nhttp://two.com\n"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestCategoryEditSaveActionDifferentPageUsesGlobalCategoryIndex(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	original := "Category: First\nhttp://one.com one\nCategory: Second\nhttp://two.com two\nPage\nCategory: Third\nhttp://three.com three\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}

	newSection := "Category: Third\nhttp://changed.com changed"
	form := url.Values{"text": {newSection}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}, "tab": {"0"}, "page": {"1"}}
	req := httptest.NewRequest("POST", "/editCategory?index=2", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryEditSaveAction(w, req); err != nil {
		t.Fatalf("CategoryEditSaveAction: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	expected := "Category: First\nhttp://one.com one\nCategory: Second\nhttp://two.com two\nPage\n" + newSection
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestCategoryEditSaveActionConcurrentReturnsEditConflict(t *testing.T) {
	p, user, _, ctx := setupCategoryEditTest(t)
	original := "Category: First\nhttp://one.com one\nCategory: Second\nhttp://two.com two\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", original); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha1, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	updated := "Category: First\nhttp://one.com one\nCategory: Second\nhttp://updated.com updated\n"
	if err := p.UpdateBookmarks(context.Background(), user, nil, "refs/heads/main", "main", updated, sha1); err != nil {
		t.Fatalf("UpdateBookmarks: %v", err)
	}

	rejected := "Category: Second\nhttp://rejected.com rejected"
	form := url.Values{"text": {rejected}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha1}, "tab": {"0"}, "page": {"0"}}
	req := httptest.NewRequest("POST", "/editCategory?edit=1&index=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := CategoryEditSaveAction(w, req); err != ErrHandled {
		t.Fatalf("expected handled conflict, got %v", err)
	}
	if w.Code != 409 {
		t.Fatalf("expected HTTP 409, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Save rejected") || !strings.Contains(body, "http://updated.com updated") || !strings.Contains(body, "http://rejected.com rejected") {
		t.Fatalf("expected conflict response with current and rejected category, got %q", body)
	}

	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks after: %v", err)
	}
	if got != updated {
		t.Fatalf("bookmarks changed unexpectedly: %q", got)
	}
}
