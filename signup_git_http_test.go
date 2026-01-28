package gobookmarks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestGitSignupScenario(t *testing.T) {
	tmp := t.TempDir()

	cfg := NewConfiguration(Config{
		LocalGitPath: tmp,
		SessionKey:   "secret",
	})
	cfg.SessionName = "testsession"

	// signup
	form := url.Values{"username": {"alice"}, "password": {"secret"}}
	req := httptest.NewRequest("POST", "/signup/git", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := context.WithValue(req.Context(), ContextValues("configuration"), cfg)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	if err := GitSignupAction(w, req); err != nil {
		t.Fatalf("signup action: %v", err)
	}
	if _, err := os.Stat(passwordPath("alice", tmp)); err != nil {
		t.Fatalf("password not created: %v", err)
	}

	p := GitProvider{}
	if ok, err := p.RepoExists(ctx, "alice", nil, "MyBookmarks"); err != nil || !ok {
		// RepoName defaults to "MyBookmarks" in GetRepoName()
		t.Fatalf("repo exists: %v %v", ok, err)
	}
	got, _, err := p.GetBookmarks(ctx, "alice", "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("get bookmarks: %v", err)
	}
	if got != defaultBookmarks {
		t.Fatalf("bookmarks mismatch")
	}

	// login
	req = httptest.NewRequest("POST", "/login/git", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx) // reuse context with config
	w = httptest.NewRecorder()
	if err := GitLoginAction(w, req); err != nil {
		t.Fatalf("login action: %v", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("no session cookie")
	}
	// obtain session from cookie for further actions
	sessReq := httptest.NewRequest("GET", "/", nil)
	sessReq.AddCookie(cookies[len(cookies)-1])
	sessW := httptest.NewRecorder()
	session, err := getSession(sessW, sessReq, cfg)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	ctx = context.WithValue(sessReq.Context(), ContextValues("session"), session)
	ctx = context.WithValue(ctx, ContextValues("provider"), "git")
	ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{})
	ctx = context.WithValue(ctx, ContextValues("configuration"), cfg)

	// create bookmarks on new branch
	createText := "Category: New\nhttp://example.com new"
	createForm := url.Values{"text": {createText}, "branch": {"feature"}}
	req = httptest.NewRequest("POST", "/edit/create", strings.NewReader(createForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	if err := BookmarksEditCreateAction(w, req); err != nil {
		t.Fatalf("create action: %v", err)
	}
	got, _, err = p.GetBookmarks(ctx, "alice", "refs/heads/feature", nil)
	if err != nil {
		t.Fatalf("get feature bookmarks: %v", err)
	}
	if got != createText {
		t.Fatalf("created bookmarks mismatch")
	}

	// update bookmarks on main branch
	updated := "Category: Updated\nhttp://example.com updated"
	_, sha, err := p.GetBookmarks(ctx, "alice", "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("get bookmarks sha: %v", err)
	}
	saveForm := url.Values{"text": {updated}, "branch": {"main"}, "ref": {"refs/heads/main"}, "sha": {sha}}
	req = httptest.NewRequest("POST", "/edit/save", strings.NewReader(saveForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	if err := BookmarksEditSaveAction(w, req); err != nil {
		t.Fatalf("save action: %v", err)
	}
	got, _, err = p.GetBookmarks(ctx, "alice", "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("get updated bookmarks: %v", err)
	}
	if got != updated {
		t.Fatalf("updated bookmarks mismatch")
	}

	// logout
	req = httptest.NewRequest("POST", "/logout", nil)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	if err := UserLogoutAction(w, req); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, ok := session.Values["GithubUser"]; ok {
		t.Fatalf("user not cleared")
	}
}

func TestGitLoginIgnoresInvalidSession(t *testing.T) {
	tmp := t.TempDir()

	cfg := NewConfiguration(Config{
		LocalGitPath: tmp,
		SessionKey:   "secret",
	})
	cfg.SessionName = "testsession"
	version = "vtest"

	// create user
	p := GitProvider{}
	ctx := context.WithValue(context.Background(), ContextValues("configuration"), cfg)
	if err := p.CreateUser(ctx, "alice", "secret"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	form := url.Values{"username": {"alice"}, "password": {"secret"}}
	req := httptest.NewRequest("POST", "/login/git", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "testsession", Value: "invalid"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	if err := GitLoginAction(w, req); err != nil {
		t.Fatalf("login action: %v", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie")
	}
	if cookies[len(cookies)-1].Value == "invalid" {
		t.Fatalf("session cookie not replaced")
	}
}
