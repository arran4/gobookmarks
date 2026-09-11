package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	gb "github.com/arran4/gobookmarks"
)

type mockRoundTripperBrowser struct{}

func (m *mockRoundTripperBrowser) RoundTrip(req *http.Request) (*http.Response, error) {
	body := "{\"access_token\": \"mock-token\", \"token_type\": \"bearer\"}"

	if strings.Contains(req.URL.Path, "user") {
		// mock user
		body = "{\"login\": \"mockuser\"}"
	}

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     header,
	}, nil
}

func TestBrowserSessionLifecycleRegression(t *testing.T) {
	gb.Config.SessionName = "gobookmarks_test_session"
	gb.Config.GithubClientID = "mock_client"
	gb.Config.GithubSecret = "mock_secret"
	gb.SessionStore = gb.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))
	gb.Config.ExternalURL = "https://example.com"
	gb.Config.ProviderOrder = []string{"github"}
	gb.SetProviderOrder(gb.Config.ProviderOrder)

	gitP := gb.GitProvider{}
	gb.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), "mockuser", nil, gb.Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	p := &gb.GitHubProvider{}
	gb.RegisterProvider(p)

	gb.Config.GithubClientID = "mock_client"
	gb.Config.GithubSecret = "mock_secret"
	gb.Config.ExternalURL = "https://example.com"
	gb.Config.SessionName = "gobookmarks_test_session"
	gb.SessionStore = gb.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))
	gb.Config.ProviderOrder = []string{"github"}
	gb.SetProviderOrder([]string{"github"})

	// Ensure provider map is populated for ConfiguredProviderNames
	// (GitHubProvider registers itself inside init())

	// Ensure we intercept the HTTP client within the actual router
	http.DefaultTransport = &mockRoundTripperBrowser{}

	r := setupRouter()
	registerRoutes(r)

	browser, err := NewBrowser(r, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}

	resp1, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	// /tab/2 without auth doesn't fail with 401, it just doesn't show authenticated data or sets session
	// The requirement is: "A clean anonymous GET /tab/2 must not emit an unnecessary deletion cookie solely because the session lacks a version."
	if len(resp1.Cookies) > 0 {
		t.Fatalf("Anonymous request should not emit cookies merely due to missing version. Got: %v", resp1.Cookies)
	}

	// 2. Initiate Login
	resp2, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Response.StatusCode != http.StatusTemporaryRedirect && resp2.Response.StatusCode != http.StatusFound {
		t.Fatalf("Expected /login to redirect, got %d", resp2.Response.StatusCode)
	}
	loc, _ := resp2.Response.Location()

	if len(resp2.Cookies) == 0 {
		t.Fatalf("Expected OAuth pending state cookie to be emitted")
	}

	stateParam := loc.Query().Get("state")

	resp3, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(stateParam)+"&code=mockcode", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp3.Response.StatusCode != http.StatusFound && resp3.Response.StatusCode != http.StatusTemporaryRedirect && resp3.Response.StatusCode != http.StatusSeeOther {
		t.Fatalf("Expected successful callback to redirect, got %d", resp3.Response.StatusCode)
	}

	loc3, _ := resp3.Response.Location()
	if loc3.Path != "/tab/2" {
		t.Fatalf("Expected post-login redirect to /tab/2, got %v", loc3.String())
	}

	// Validate ordered Set-Cookie assertion
	foundLive := false
	cookieMap := make(map[string]bool) // name:path:domain -> isLive

	for _, cookie := range resp3.Cookies {
		id := cookie.Name + ":" + cookie.Path + ":" + cookie.Domain
		isLive := cookie.Expires.IsZero() || cookie.Expires.After(time.Now())
		cookieMap[id] = isLive
	}

	for id, isLive := range cookieMap {
		if id == gb.Config.GetSessionName()+":/:" {
			foundLive = isLive
		}
	}

	if !foundLive {
		t.Fatalf("Callback cookie regression: Set-Cookie sequence ended with an expired cookie. Mutations: %v", resp3.Cookies)
	}

	resp4, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp4.Response.StatusCode != http.StatusOK {
		t.Fatalf("Expected /tab/2 to be 200 OK after login, got %d", resp4.Response.StatusCode)
	}

	bodyData, _ := io.ReadAll(resp4.Response.Body)
	if !bytes.Contains(bodyData, []byte("Logout")) {
		t.Fatalf("Expected authenticated page to contain Logout button")
	}
}
