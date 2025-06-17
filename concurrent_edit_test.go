package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
)

func TestBookmarksEditSaveConcurrent(t *testing.T) {
	tmp := t.TempDir()
	LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	if err := p.CreateRepo(context.Background(), user, nil, RepoName); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	initial := "Category: A\n"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", initial); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	_, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	if err := p.UpdateBookmarks(context.Background(), user, nil, "refs/heads/main", "main", "Category: B\n", sha); err != nil {
		t.Fatalf("UpdateBookmarks: %v", err)
	}

	SessionName = "testsession"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	sessReq := httptest.NewRequest("GET", "/", nil)
	sess, _ := getSession(httptest.NewRecorder(), sessReq)
	sess.Values["GithubUser"] = &User{Login: user}
	ctx := context.WithValue(sessReq.Context(), ContextValues("session"), sess)
	ctx = context.WithValue(ctx, ContextValues("provider"), "git")

	form := url.Values{"text": {"Category: C"}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}}
	req := httptest.NewRequest("POST", "/edit/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	err = BookmarksEditSaveAction(w, req)
	if err == nil || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("expected concurrent error, got %v", err)
	}
}
