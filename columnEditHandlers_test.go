package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEditColumnPostAction(t *testing.T) {
	bookmarksStr := "Tab: Home\nPage: Main\nColumn: Alpha\nCategory: Links\nhttp://example.com"
	req := httptest.NewRequest("POST", "/editColumn", strings.NewReader(url.Values{
		"tab":    {"0"},
		"page":   {"0"},
		"block":  {"0"},
		"col":    {"0"},
		"name":   {"Beta"},
		"sha":    {"test-sha"},
		"ref":    {"refs/heads/source"},
		"branch": {"test-branch"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := req.Context()
	cd := &CoreData{}
	ctx = context.WithValue(ctx, ContextValues("coreData"), cd)

	Config = Configuration{}
	session, _ := InitSessionStore([]byte("test")).Get(req, "gobookmarks")
	session.Values["GithubUser"] = &User{Login: "testuser"}
	ctx = context.WithValue(ctx, ContextValues("session"), session)

	var gotGetRef, gotSourceRef, gotBranch string
	mockProvider := &mockAccessProvider{nameFunc: func() string { return "mock" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			gotGetRef = ref
			return bookmarksStr, "test-sha", nil
		},
	}
	var savedData string
	mockProvider.updateBookmarksFunc = func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
		savedData = text
		gotSourceRef = sourceRef
		gotBranch = branch
		return nil
	}

	ctx = context.WithValue(ctx, ContextValues("provider"), "mock")

	RegisterProvider(mockProvider)

	cd.requestCache = &requestCache{data: make(map[string]*bookmarkCacheEntry)}

	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	err := EditColumnPostAction(w, req)

	if err != ErrHandled {
		t.Fatalf("Expected ErrHandled redirect, got %v", err)
	}

	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected status See Other, got %d", w.Code)
	}

	if mockProvider.updateBookmarksCalls != 1 {
		t.Fatalf("Expected 1 save call, got %d", mockProvider.updateBookmarksCalls)
	}

	if !strings.Contains(savedData, "Column: Beta") {
		t.Errorf("Expected 'Column: Beta' in saved data, got:\n%s", savedData)
	}

	if gotGetRef != "refs/heads/source" {
		t.Errorf("Expected get ref refs/heads/source, got %s", gotGetRef)
	}
	if gotSourceRef != "refs/heads/source" {
		t.Errorf("Expected source ref refs/heads/source, got %s", gotSourceRef)
	}
	if gotBranch != "test-branch" {
		t.Errorf("Expected branch 'test-branch', got %q", gotBranch)
	}
}

func TestEditColumnPostAction_StaleSHA(t *testing.T) {
	bookmarksStr := "Tab: Home\nPage: Main\nColumn: Alpha\nCategory: Links\nhttp://example.com"
	req := httptest.NewRequest("POST", "/editColumn", strings.NewReader(url.Values{
		"tab":    {"0"},
		"page":   {"0"},
		"block":  {"0"},
		"col":    {"0"},
		"name":   {"Beta"},
		"sha":    {"stale-sha"},
		"ref":    {"main"},
		"branch": {"test-branch"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := req.Context()
	cd := &CoreData{}
	ctx = context.WithValue(ctx, ContextValues("coreData"), cd)

	session, _ := InitSessionStore([]byte("test")).Get(req, "gobookmarks")
	session.Values["GithubUser"] = &User{Login: "testuser"}
	ctx = context.WithValue(ctx, ContextValues("session"), session)

	mockProvider := &mockAccessProvider{nameFunc: func() string { return "mock" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return bookmarksStr, "actual-sha", nil
		},
	}
	RegisterProvider(mockProvider)
	ctx = context.WithValue(ctx, ContextValues("provider"), "mock")
	cd.requestCache = &requestCache{data: make(map[string]*bookmarkCacheEntry)}
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	err := EditColumnPostAction(w, req)

	if err == nil || !strings.Contains(err.Error(), "concurrent modification detected") {
		t.Fatalf("Expected concurrent modification error, got %v", err)
	}
}

func TestEditColumnPostAction_UnnamedToNamed(t *testing.T) {
	bookmarksStr := "Tab: Home\nPage: Main\nColumn\nCategory: Links\nhttp://example.com"
	req := httptest.NewRequest("POST", "/editColumn", strings.NewReader(url.Values{
		"tab":    {"0"},
		"page":   {"0"},
		"block":  {"0"},
		"col":    {"0"},
		"name":   {"AcquiredName"},
		"sha":    {"test-sha"},
		"ref":    {"main"},
		"branch": {"main"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := req.Context()
	cd := &CoreData{}
	ctx = context.WithValue(ctx, ContextValues("coreData"), cd)

	session, _ := InitSessionStore([]byte("test")).Get(req, "gobookmarks")
	session.Values["GithubUser"] = &User{Login: "testuser"}
	ctx = context.WithValue(ctx, ContextValues("session"), session)

	mockProvider := &mockAccessProvider{nameFunc: func() string { return "mock" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return bookmarksStr, "test-sha", nil
		},
	}
	var savedData string
	mockProvider.updateBookmarksFunc = func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
		savedData = text
		return nil
	}

	ctx = context.WithValue(ctx, ContextValues("provider"), "mock")

	RegisterProvider(mockProvider)

	cd.requestCache = &requestCache{data: make(map[string]*bookmarkCacheEntry)}

	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	err := EditColumnPostAction(w, req)

	if err != ErrHandled {
		t.Fatalf("Expected ErrHandled redirect, got %v", err)
	}

	if !strings.Contains(savedData, "Column: AcquiredName") {
		t.Errorf("Expected 'Column: AcquiredName' in saved data, got:\n%s", savedData)
	}
}

func TestEditColumnPostAction_NamedToUnnamed(t *testing.T) {
	bookmarksStr := "Tab: Home\nPage: Main\nColumn: OldName\nCategory: Links\nhttp://example.com"
	req := httptest.NewRequest("POST", "/editColumn", strings.NewReader(url.Values{
		"tab":    {"0"},
		"page":   {"0"},
		"block":  {"0"},
		"col":    {"0"},
		"name":   {"  "}, // whitespace gets trimmed to empty -> unnamed
		"sha":    {"test-sha"},
		"ref":    {"main"},
		"branch": {"main"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := req.Context()
	cd := &CoreData{}
	ctx = context.WithValue(ctx, ContextValues("coreData"), cd)

	session, _ := InitSessionStore([]byte("test")).Get(req, "gobookmarks")
	session.Values["GithubUser"] = &User{Login: "testuser"}
	ctx = context.WithValue(ctx, ContextValues("session"), session)

	mockProvider := &mockAccessProvider{nameFunc: func() string { return "mock" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return bookmarksStr, "test-sha", nil
		},
	}
	var savedData string
	mockProvider.updateBookmarksFunc = func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
		savedData = text
		return nil
	}

	ctx = context.WithValue(ctx, ContextValues("provider"), "mock")

	RegisterProvider(mockProvider)

	cd.requestCache = &requestCache{data: make(map[string]*bookmarkCacheEntry)}

	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	err := EditColumnPostAction(w, req)

	if err != ErrHandled {
		t.Fatalf("Expected ErrHandled redirect, got %v", err)
	}

	if strings.Contains(savedData, "Column: ") {
		t.Errorf("Expected bare 'Column' in saved data, got:\n%s", savedData)
	}
}
