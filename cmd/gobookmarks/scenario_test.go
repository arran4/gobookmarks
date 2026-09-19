package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
	"golang.org/x/oauth2"
)

func TestParseScenario(t *testing.T) {
	txtar := `Test preamble
-- scenario.meta --
Name: test

-- 01-user.event --
Op: user.create
Username: testuser
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	if s.Preamble != "Test preamble\n" {
		t.Errorf("expected preamble 'Test preamble\n', got %q", s.Preamble)
	}

	if s.Manifest["Name"] != "test" {
		t.Errorf("expected manifest Name 'test', got %q", s.Manifest["Name"])
	}

	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.Events))
	}

	if s.Events[0].Op != "user.create" {
		t.Errorf("expected event Op 'user.create', got %q", s.Events[0].Op)
	}
}

func TestScenarioValidation(t *testing.T) {
	txtar := `
-- 01-user.event --
Op: user.create
` // Missing Username

	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	err = ValidateScenario(s)
	if err == nil {
		t.Errorf("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "missing Username") {
		t.Errorf("expected missing Username error, got: %v", err)
	}
}

func TestBookmarkAssetResolution(t *testing.T) {
	txtar := `
-- 001-user.event --
Op: user.create
Ref: user
Username: user

-- 010-bookmarks.event --
Op: bookmark.create
User: user
Asset: missing.txt
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}
	err = ValidateScenario(s)
	if err == nil {
		t.Errorf("expected validation failure due to missing asset")
	}
	if !strings.Contains(err.Error(), "missing asset: missing.txt") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBookmarkAssetResolutionSuccess(t *testing.T) {
	txtar := `
-- 01-user.event --
Op: user.create
Ref: alice
Username: alice

-- 02-bookmarks.event --
Op: bookmark.create
User: alice
Asset: bookmarks/alice.txt

-- bookmarks/alice.txt --
https://example.com Example
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScenario(s); err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if s.Files["bookmarks/alice.txt"] == "" {
		t.Fatal("bookmark asset was not retained")
	}
}

func TestScenarioManifestAndReferenceValidation(t *testing.T) {
	for _, tc := range []struct{ name, scenario, want string }{
		{"unsupported storage", "-- scenario.meta --\nStorageProvider: missing\n", "unsupported StorageProvider"},
		{"unresolved ref", "-- a.event --\nOp: repo.create\nUser: nobody\nName: bookmarks\n", "unresolved user ref"},
		{"duplicate ref", "-- a.event --\nOp: user.create\nRef: user\nUsername: a\n-- b.event --\nOp: user.create\nRef: user\nUsername: b\n", "duplicate ref"},
		{"provider conflict", "-- scenario.meta --\nStorageProvider: sql\n-- a.event --\nOp: user.create\nUsername: a\nProvider: git\n", "conflicts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ParseScenario(strings.NewReader(tc.scenario))
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateScenario(s); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateScenario error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestUnknownOperation(t *testing.T) {
	txtar := `
-- 010-unknown.event --
Op: unknown.op
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}
	err = ValidateScenario(s)
	if err == nil {
		t.Errorf("expected error for unknown op")
	}
}

func TestSymbolicReferenceFailure(t *testing.T) {
	txtar := `
-- 010-repo.event --
Op: repo.create
User: missing-ref
Name: main
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	sCtx := &ScenarioContext{
		Refs:            map[string]string{},
		Files:           s.Files,
		StorageProvider: "sql",
	}

	op := operations["repo.create"]
	err = op.Apply(context.Background(), sCtx, s.Events[0])
	if err == nil {
		t.Errorf("expected unknown user ref error")
	}
	if !strings.Contains(err.Error(), "unknown user ref") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTempSQLiteBackend(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}
}

func TestHistoryScenario(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	scenarioPath := "scenarios/history.txtar"
	file, err := os.Open(scenarioPath)
	if err != nil {
		t.Fatalf("Failed to open scenario %s: %v", scenarioPath, err)
	}
	defer func() { _ = file.Close() }()

	scenario, err := ParseScenario(file)
	if err != nil {
		t.Fatalf("Failed to parse scenario: %v", err)
	}

	if err := ValidateScenario(scenario); err != nil {
		t.Fatalf("Failed to validate scenario: %v", err)
	}

	applyCtx := context.WithValue(context.Background(), gobookmarks.ContextValues("provider"), "sql")
	if err := ApplyScenario(applyCtx, scenario); err != nil {
		t.Fatalf("Failed to apply scenario: %v", err)
	}

	// Verify history via provider
	p := gobookmarks.GetProvider("sql")
	commits, err := p.GetCommits(context.Background(), "bob", nil, "main", 1, 10)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	// We expect 3 commits from the history scenario
	if len(commits) != 3 {
		t.Errorf("Expected 3 commits, got %d", len(commits))
	}
	bookmarks, _, err := gobookmarks.GetBookmarks(applyCtx, "bob", "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks through access layer: %v", err)
	}
	if !strings.Contains(bookmarks, "Version 3") {
		t.Fatalf("access layer did not return the latest seeded bookmarks: %q", bookmarks)
	}

	// Verify history via the actual router logic
	r := newApplicationRouter()

	browser, err := NewBrowser(r, "http://localhost")
	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}

	login, err := browser.DoForm("/login/sql", url.Values{
		"username": {"bob"},
		"password": {"password"}, // UserCreateOp currently defaults to "password"
	})
	if err != nil {
		t.Fatalf("Failed to execute login post: %v", err)
	}
	if login.Response.StatusCode != http.StatusSeeOther {
		t.Fatalf("SQL login status = %d, location=%q, cookies=%+v", login.Response.StatusCode, login.Response.Header.Get("Location"), cookieMetadata(login.Cookies))
	}
	if location := login.Response.Header.Get("Location"); location == "/login/sql?error=invalid" || location == "" {
		t.Fatalf("SQL login failed: location=%q, cookies=%+v", location, cookieMetadata(login.Cookies))
	}

	resp, err := browser.Do("GET", "/history", nil)
	if err != nil {
		t.Fatalf("GET /history failed: %v", err)
	}

	body, readErr := io.ReadAll(resp.Response.Body)
	if readErr != nil {
		t.Fatalf("read /history response: %v", readErr)
	}
	if resp.Response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Logout")) || !bytes.Contains(body, []byte("refs/heads/main")) {
		t.Fatalf("/history status=%d location=%q cookies=%+v body=%q", resp.Response.StatusCode, resp.Response.Header.Get("Location"), cookieMetadata(resp.Cookies), body)
	}
	latest, err := browser.Do("GET", "/?ref="+url.QueryEscape(commits[0].SHA), nil)
	if err != nil {
		t.Fatalf("GET latest revision: %v", err)
	}
	latestBody, err := io.ReadAll(latest.Response.Body)
	if err != nil {
		t.Fatalf("read latest revision: %v", err)
	}
	if latest.Response.StatusCode != http.StatusOK || !bytes.Contains(latestBody, []byte("Logout")) || !bytes.Contains(latestBody, []byte("Version 3")) {
		t.Fatalf("latest revision status=%d location=%q cookies=%+v body=%q", latest.Response.StatusCode, latest.Response.Header.Get("Location"), cookieMetadata(latest.Cookies), latestBody)
	}
}

func TestComplexBookmarksUI(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend: %v", err)
	}
	defer cleanup()

	file, err := os.Open("scenarios/complex-bookmarks.txtar")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	scenario, err := ParseScenario(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScenario(scenario); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), gobookmarks.ContextValues("provider"), "sql")
	if err := ApplyScenario(ctx, scenario); err != nil {
		t.Fatal(err)
	}

	p := gobookmarks.GetProvider("sql")
	raw, _, err := p.GetBookmarks(context.Background(), "testuser", "refs/heads/main", nil)
	if err != nil || !strings.Contains(raw, "Google") {
		t.Fatalf("direct SQL seeded data err=%v body=%q", err, raw)
	}
	throughAccess, _, err := gobookmarks.GetBookmarks(ctx, "testuser", "refs/heads/main", nil)
	if err != nil || !strings.Contains(throughAccess, "Google") {
		t.Fatalf("access-layer seeded data err=%v body=%q", err, throughAccess)
	}

	r := newApplicationRouter()
	browser, err := NewBrowser(r, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	login, err := browser.DoForm("/login/sql", url.Values{"username": {"testuser"}, "password": {"password"}})
	if err != nil {
		t.Fatal(err)
	}
	if login.Response.StatusCode != http.StatusSeeOther || login.Response.Header.Get("Location") == "/login/sql?error=invalid" {
		t.Fatalf("SQL login status=%d location=%q cookies=%+v", login.Response.StatusCode, login.Response.Header.Get("Location"), cookieMetadata(login.Cookies))
	}
	response, err := browser.Do("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Logout")) || !bytes.Contains(body, []byte("Google")) || !bytes.Contains(body, []byte("Hacker News")) {
		t.Fatalf("GET / status=%d location=%q cookies=%+v body=%q", response.Response.StatusCode, response.Response.Header.Get("Location"), cookieMetadata(response.Cookies), body)
	}
}

func TestLocalGitScenario(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	s, err := ParseScenario(strings.NewReader(`
-- scenario.meta --
StorageProvider: git
-- 01-user.event --
Op: user.create
Ref: alice
Username: alice
-- 02-repo.event --
Op: repo.create
Ref: repo
User: alice
Name: bookmarks
-- 03-v1.event --
Op: bookmark.create
User: alice

https://example.com Version 1
-- 04-v2.event --
Op: bookmark.create
User: alice

https://example.com Version 2
`))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScenario(s); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), gobookmarks.ContextValues("provider"), "git")
	if err := ApplyScenario(ctx, s); err != nil {
		t.Fatal(err)
	}
	p := gobookmarks.GetProvider("git")
	bookmarks, _, err := p.GetBookmarks(ctx, "alice", "refs/heads/main", nil)
	if err != nil || !strings.Contains(bookmarks, "Version 2") {
		t.Fatalf("git bookmarks err=%v body=%q", err, bookmarks)
	}
	commits, err := p.GetCommits(ctx, "alice", nil, "refs/heads/main", 1, 10)
	if err != nil || len(commits) < 3 { // init plus two persisted bookmark revisions
		t.Fatalf("git history err=%v commits=%d", err, len(commits))
	}
	entries, err := os.ReadDir(gobookmarks.Config.LocalGitPath)
	if err != nil || len(entries) == 0 {
		t.Fatalf("local git repository was not created under scenario temp path: %v", err)
	}
}

func TestProviderLoginScenario(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	file, err := os.Open("scenarios/provider-login.txtar")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	s, err := ParseScenario(file)
	if err != nil {
		t.Fatal(err)
	}
	if s.StorageProvider != "git" || s.AuthProvider != "github" || s.AuthUser != "charlie" {
		t.Fatalf("unexpected provider-login identity: storage=%q auth=%q user=%q", s.StorageProvider, s.AuthProvider, s.AuthUser)
	}
	if err := ValidateScenario(s); err != nil {
		t.Fatal(err)
	}
	if err := ApplyScenario(context.WithValue(context.Background(), gobookmarks.ContextValues("provider"), "git"), s); err != nil {
		t.Fatal(err)
	}

	gobookmarks.Config.GithubClientID = "scenario-client"
	gobookmarks.Config.GithubSecret = "scenario-secret"
	gobookmarks.Config.ExternalURL = "http://localhost"
	mockClient := &http.Client{Transport: &mockOAuthRoundTripper{fakeToken: "scenario-token", userLogin: s.AuthUser}}
	router := newApplicationRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), oauth2.HTTPClient, mockClient)))
	})
	browser, err := NewBrowser(handler, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	start, err := browser.Do("GET", "/login/github", nil)
	if err != nil {
		t.Fatal(err)
	}
	if start.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("login start status=%d location=%q", start.Response.StatusCode, start.Response.Header.Get("Location"))
	}
	state, err := start.Response.Location()
	if err != nil || state.Query().Get("state") == "" {
		t.Fatalf("login start state: location=%q err=%v", start.Response.Header.Get("Location"), err)
	}
	callback, err := browser.Do("GET", "/oauth2Callback?code=scenario&state="+url.QueryEscape(state.Query().Get("state")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if callback.Response.StatusCode != http.StatusFound && callback.Response.StatusCode != http.StatusSeeOther {
		t.Fatalf("callback status=%d location=%q cookies=%+v", callback.Response.StatusCode, callback.Response.Header.Get("Location"), cookieMetadata(callback.Cookies))
	}
	page, err := browser.Do("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(page.Response.Body)
	if err != nil || page.Response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Logout")) || !bytes.Contains(body, []byte("Mocked provider bookmark")) {
		t.Fatalf("provider page err=%v status=%d body=%q", err, page.Response.StatusCode, body)
	}
}

func cookieMetadata(cookies []*http.Cookie) []CookieMetadata {
	metadata := make([]CookieMetadata, 0, len(cookies))
	for _, cookie := range cookies {
		metadata = append(metadata, CookieMetadata{Identity: CookieIdentity{Name: cookie.Name, Domain: cookie.Domain, Path: cookie.Path}, MaxAge: cookie.MaxAge, Expires: cookie.Expires, Secure: cookie.Secure, HttpOnly: cookie.HttpOnly, SameSite: cookie.SameSite})
	}
	return metadata
}
