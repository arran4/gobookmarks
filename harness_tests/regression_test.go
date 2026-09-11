package harness_tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/arran4/gobookmarks"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

// mockRoundTripper simulates the token endpoint
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// simulate a successful token response
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

// Ensure the project gets initialized fully for testing
func TestSessionLifecycleRegression(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.Config.GithubClientID = "mock_client"
	gobookmarks.Config.GithubSecret = "mock_secret"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))

	// Create mock user repo so ensureRepo doesn't fail
	p := &gobookmarks.GitHubProvider{}
	gobookmarks.RegisterProvider(p)

	// Need to make sure GitProvider is registered for Repo checking
	gitP := gobookmarks.GitProvider{}
	gobookmarks.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), "mockuser", nil, gobookmarks.Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	// We need a router with the real middleware stack and handlers.
	// Since setupRouter() is in package main (cmd/gobookmarks) we rebuild the relevant ones here.
	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

	// Stub out the core routes needed for login
	r.HandleFunc("/tab/2", func(w http.ResponseWriter, req *http.Request) {
		session := gobookmarks.GetSession(w, req)
		user, _ := session.Values["GithubUser"].(*gobookmarks.User)
		if user != nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Authenticated as " + user.Login))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Anonymous"))
		}
	}).Methods("GET")

	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, req *http.Request) { _ = gobookmarks.LoginWithProvider(w, req) }).Methods("GET")
	r.HandleFunc("/oauth2Callback", func(w http.ResponseWriter, req *http.Request) {
		// Mock the HTTP client context
		client := &http.Client{Transport: &mockRoundTripper{}}
		ctx := context.WithValue(req.Context(), oauth2.HTTPClient, client)
		req = req.WithContext(ctx)

		err := gobookmarks.Oauth2CallbackPage(w, req)
		if err != nil {
			// Print out the specific error for debugging
			t.Logf("Oauth2CallbackPage failed: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}

		// Run redirectToHandler to emulate the real chain
		session := gobookmarks.GetSession(w, req)
		redirectUrl := "/"
		if rURL, ok := session.Values["Redirect"].(string); ok && rURL != "" {
			redirectUrl = rURL
		}
		http.Redirect(w, req, redirectUrl, http.StatusTemporaryRedirect)
	}).Methods("GET")

	browser, err := NewBrowser(r, "https://example.com")

	// Add config for proper matching
	gobookmarks.Config.ExternalURL = "https://example.com"

	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}

	// 1. Initial anonymous request
	resp1, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp1.Response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected anonymous /tab/2 to be 401, got %d", resp1.Response.StatusCode)
	}
	// Important check: Must not emit any Set-Cookie for an anonymous request that doesn't save state
	if len(resp1.Cookies) > 0 {
		t.Fatalf("Anonymous request should not emit cookies merely due to missing version. Got: %v", resp1.Cookies)
	}

	// 2. Initiate Login
	resp2, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected /login to redirect, got %d", resp2.Response.StatusCode)
	}
	loc, _ := resp2.Response.Location()

	// Must have emitted a valid cookie for OAuth pending state
	if len(resp2.Cookies) == 0 {
		t.Fatalf("Expected OAuth pending state cookie to be emitted")
	}

	// 3. Callback
	// Extract the state parameter sent to the provider
	stateParam := loc.Query().Get("state")

	// Fast-forward to callback

	resp3, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(stateParam)+"&code=mockcode", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp3.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected successful callback to redirect, got %d", resp3.Response.StatusCode)
	}

	loc3, _ := resp3.Response.Location()
	if loc3.Path != "/tab/2" {
		t.Fatalf("Expected post-login redirect to /tab/2, got %v", loc3.String())
	}

	// Inspect the callback cookies. We must not have an expired one AFTER a valid authenticated one.
	// Actually, look at the final cookie in the jar to ensure it is valid.
	foundLive := false
	// check the target url to ensure cookie matches path /tab/2
	targetURL, _ := browser.Origin.Parse("/tab/2")
	for _, cookie := range browser.Jar.Cookies(targetURL) {
		if cookie.Name == gobookmarks.Config.GetSessionName() {
			foundLive = true
		}
	}
	if !foundLive {
		t.Fatalf("Browser jar contains no live authenticated cookie after callback. Mutations were: %v", resp3.Cookies)
	}

	// 4. Follow redirect back to /tab/2
	resp4, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp4.Response.StatusCode != http.StatusOK {
		t.Fatalf("Expected /tab/2 to be 200 OK after login, got %d", resp4.Response.StatusCode)
	}
}
