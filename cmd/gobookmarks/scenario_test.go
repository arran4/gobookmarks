package main

import (
	"bytes"
	"context"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
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
	tests := []struct {
		name        string
		scenario    string
		expectError string
	}{
		{"unsupported storage", "-- scenario.meta --\nStorageProvider: missing\n", "unsupported StorageProvider"},
		{"unsupported auth", "-- scenario.meta --\nStorageProvider: sql\nAuthProvider: missing\n", "unsupported AuthProvider"},
		{"unresolved ref", "-- scenario.meta --\nStorageProvider: sql\n-- 01.event --\nOp: repo.create\nUser: unknown\nName: x\n", "unresolved user ref"},
		{"duplicate ref", "-- scenario.meta --\nStorageProvider: sql\n-- 01.event --\nOp: user.create\nRef: r1\nUsername: u1\n-- 02.event --\nOp: user.create\nRef: r1\nUsername: u2\n", "duplicate ref"},
		{"provider conflict", "-- scenario.meta --\nStorageProvider: sql\n-- 01.event --\nOp: user.create\nProvider: git\nUsername: u1\n", "conflicts"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalSql := gobookmarks.GetProvider("sql")
			originalOrder := append([]string(nil), gobookmarks.ProviderNames()...)
			t.Cleanup(func() {
				if originalSql != nil {
					gobookmarks.RegisterProvider(originalSql)
				} else {
					gobookmarks.UnregisterProvider("sql")
				}
				gobookmarks.SetProviderOrder(originalOrder)
			})

			// We only need sql for the tests testing non-storage provider validation errors, since missing is used for the first test
			gobookmarks.RegisterProvider(&gobookmarks.SQLProvider{})

			s, err := ParseScenario(strings.NewReader(tc.scenario))
			if err != nil {
				t.Fatalf("ParseScenario error = %v", err)
			}
			err = ValidateScenario(s)
			if err == nil {
				t.Fatalf("ValidateScenario error = nil, want %q", tc.expectError)
			}
			if !strings.Contains(err.Error(), tc.expectError) {
				t.Errorf("ValidateScenario error = %v, want %q", err, tc.expectError)
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
		t.Fatal(err)
	}
	err = ValidateScenario(s)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unknown operation: unknown.op") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSymbolicReferenceFailure(t *testing.T) {
	s := &ScenarioContext{Refs: make(map[string]string), StorageProvider: "sql"}

	originalSql := gobookmarks.GetProvider("sql")
	originalOrder := append([]string(nil), gobookmarks.ProviderNames()...)
	t.Cleanup(func() {
		if originalSql != nil {
			gobookmarks.RegisterProvider(originalSql)
		} else {
			gobookmarks.UnregisterProvider("sql")
		}
		gobookmarks.SetProviderOrder(originalOrder)
	})

	// We need sql for this test
	gobookmarks.RegisterProvider(&gobookmarks.SQLProvider{})

	op := &RepoCreateOp{}
	e := &Event{Props: map[string]string{"User": "missing_ref", "Name": "bookmarks"}}
	err := op.Apply(context.Background(), s, e)
	if err == nil {
		t.Fatal("expected error for missing reference")
	}
	if !strings.Contains(err.Error(), "unknown user ref") {
		t.Fatalf("unexpected error: %v", err)
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

	if gobookmarks.Config.DBConnectionProvider != "sqlite3" {
		t.Errorf("expected DB provider sqlite3, got %q", gobookmarks.Config.DBConnectionProvider)
	}

	if !strings.Contains(gobookmarks.Config.DBConnectionString, "mode=memory") {
		t.Errorf("expected DB string to contain mode=memory, got %q", gobookmarks.Config.DBConnectionString)
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

	configureScenarioAuth(s)
	gobookmarks.Config.ExternalURL = "http://localhost"
	mockClient := &http.Client{Transport: &mockOAuthRoundTripper{fakeToken: "scenario-token", userLogin: s.AuthUser}}
	handler := scenarioApplicationHandler(s, mockClient)
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
	if err != nil || page.Response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Logout")) || !strings.Contains(html.UnescapeString(string(body)), "Charlie's Example") {
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
