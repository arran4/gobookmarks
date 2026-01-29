package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/arran4/gobookmarks/core"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func setupCategoryEditTest(t *testing.T) (GitProvider, string, *sessions.Session, context.Context) {
	tmp := t.TempDir()
	LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	if err := p.CreateRepo(context.Background(), user, nil, RepoName); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	SessionName = "testsession"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	sessReq := httptest.NewRequest("GET", "/", nil)
	sess, err := getSession(httptest.NewRecorder(), sessReq)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	sess.Values["GithubUser"] = &core.BasicUser{Login: user}
	sess.Values["Token"] = &oauth2.Token{}
	ctx := context.WithValue(sessReq.Context(), core.ContextValues("session"), sess)
	ctx = context.WithValue(ctx, core.ContextValues("provider"), "git")
	ctx = context.WithValue(ctx, core.ContextValues("coreData"), &core.CoreData{Session: sess})
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
