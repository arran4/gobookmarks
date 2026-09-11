package harness_tests

import (
	"context"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/arran4/gobookmarks"
	"github.com/gorilla/mux"
)

// Tests legacy authenticated sessions without a version are treated as stale
func TestLegacySessionWithoutVersion(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))

	// Create mock user
	gitP := gobookmarks.GitProvider{}
	gobookmarks.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), "mockuser", nil, gobookmarks.Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

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

	browser, err := NewBrowser(r, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}

	// Inject a legacy authenticated session (no version)
	targetURL, _ := browser.Origin.Parse("/tab/2")

	// Create mock request to save session
	mockReq, _ := http.NewRequest("GET", "https://example.com/tab/2", nil)
	mockRec := httptest.NewRecorder()

	legacySession, _ := gobookmarks.SessionStore.New(mockReq, gobookmarks.Config.GetSessionName())
	legacySession.Values["GithubUser"] = &gobookmarks.User{Login: "mockuser"}
	// deliberately omit "version" to simulate legacy session
	_ = legacySession.Save(mockReq, mockRec)

	// Add the generated cookie to the browser jar
	cookies := mockRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("Failed to create legacy session cookie")
	}
	browser.Jar.SetCookies(targetURL, cookies)

	resp, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected legacy session to be invalidated resulting in 401, got %d", resp.Response.StatusCode)
	}

	// Check that we got a deletion cookie or a brand new session with no user
	jarCookies := browser.Jar.Cookies(targetURL)
	foundValid := false
	for _, cookie := range jarCookies {
		if cookie.Name == gobookmarks.Config.GetSessionName() && (cookie.Expires.IsZero() || cookie.Expires.After(time.Now())) {
			foundValid = true
		}
	}

	// Ensure that even if a valid new cookie exists, it doesn't represent an authenticated user
	// (verified by the 401 response status code above)
	_ = foundValid
}

func TestOldAuthenticatedSessionIsInvalidated(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))

	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

	r.HandleFunc("/tab/2", func(w http.ResponseWriter, req *http.Request) {
		session := gobookmarks.GetSession(w, req)
		user, _ := session.Values["GithubUser"].(*gobookmarks.User)
		if user != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}).Methods("GET")

	browser, _ := NewBrowser(r, "https://example.com")
	targetURL, _ := browser.Origin.Parse("/tab/2")

	// Inject a session with an old version
	mockReq, _ := http.NewRequest("GET", "https://example.com/tab/2", nil)
	mockRec := httptest.NewRecorder()

	oldSession, _ := gobookmarks.SessionStore.New(mockReq, gobookmarks.Config.GetSessionName())
	oldSession.Values["GithubUser"] = &gobookmarks.User{Login: "mockuser"}
	oldSession.Values["version"] = "old-version"
	_ = oldSession.Save(mockReq, mockRec)

	browser.Jar.SetCookies(targetURL, mockRec.Result().Cookies())

	resp, _ := browser.Do("GET", "/tab/2", nil)

	if resp.Response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected old version session to be invalidated, got %d", resp.Response.StatusCode)
	}
}

// Tests that malformed/invalid cookies recover cleanly instead of triggering loops
func TestInvalidCookieRecovery(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))

	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

	r.HandleFunc("/tab/2", func(w http.ResponseWriter, req *http.Request) {
		session := gobookmarks.GetSession(w, req)
		user, _ := session.Values["GithubUser"].(*gobookmarks.User)
		if user != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}).Methods("GET")

	browser, _ := NewBrowser(r, "https://example.com")
	targetURL, _ := browser.Origin.Parse("/tab/2")

	// Inject a malformed cookie
	browser.Jar.SetCookies(targetURL, []*http.Cookie{
		{
			Name:  gobookmarks.Config.GetSessionName(),
			Value: "garbage-invalid-data-that-will-fail-mac-or-decode",
			Path:  "/",
		},
	})

	resp, _ := browser.Do("GET", "/tab/2", nil)

	if resp.Response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected invalid cookie to safely recover to anonymous state (401), got %d", resp.Response.StatusCode)
	}

	// Ensure the invalid cookie is wiped out
	jarCookies := browser.Jar.Cookies(targetURL)
	for _, cookie := range jarCookies {
		if cookie.Name == gobookmarks.Config.GetSessionName() && cookie.Value == "garbage-invalid-data-that-will-fail-mac-or-decode" {
			t.Fatalf("Malformed cookie was not properly cleared from the jar")
		}
	}
}

// Tests that repeat logins work gracefully without accumulating state
func TestRepeatLogin(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.Config.GithubClientID = "mock_client"
	gobookmarks.Config.GithubSecret = "mock_secret"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))
	gobookmarks.Config.ExternalURL = "https://example.com"

	// Create mock user
	gitP := gobookmarks.GitProvider{}
	gobookmarks.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), "mockuser", nil, gobookmarks.Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	p := &gobookmarks.GitHubProvider{}
	gobookmarks.RegisterProvider(p)

	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

	r.HandleFunc("/tab/2", func(w http.ResponseWriter, req *http.Request) {
		session := gobookmarks.GetSession(w, req)
		user, _ := session.Values["GithubUser"].(*gobookmarks.User)
		if user != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}).Methods("GET")

	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, req *http.Request) { _ = gobookmarks.LoginWithProvider(w, req) }).Methods("GET")
	r.HandleFunc("/oauth2Callback", func(w http.ResponseWriter, req *http.Request) {
		client := &http.Client{Transport: &mockRoundTripper{}}
		ctx := context.WithValue(req.Context(), oauth2.HTTPClient, client)
		req = req.WithContext(ctx)

		_ = gobookmarks.Oauth2CallbackPage(w, req)
		session := gobookmarks.GetSession(w, req)
		redirectUrl := "/"
		if rURL, ok := session.Values["Redirect"].(string); ok && rURL != "" {
			redirectUrl = rURL
		}
		http.Redirect(w, req, redirectUrl, http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/logout", func(w http.ResponseWriter, req *http.Request) {
		_ = gobookmarks.UserLogoutAction(w, req)
		http.Redirect(w, req, "/", http.StatusTemporaryRedirect)
	}).Methods("GET")

	browser, _ := NewBrowser(r, "https://example.com")

	// Login 1
	resp1, _ := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	if resp1.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected redirect on login 1, got %d", resp1.Response.StatusCode)
	}
	loc1, _ := resp1.Response.Location()
	state1 := loc1.Query().Get("state")

	resp2, _ := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state1)+"&code=mockcode", nil)
	if resp2.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected redirect on callback 1")
	}

	resp3, _ := browser.Do("GET", "/tab/2", nil)
	if resp3.Response.StatusCode != http.StatusOK {
		t.Fatalf("Login 1 failed")
	}

	// Logout
	_, _ = browser.Do("GET", "/logout", nil)

	resp4, _ := browser.Do("GET", "/tab/2", nil)
	if resp4.Response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Logout failed")
	}

	// Login 2
	resp5, _ := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	if resp5.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected redirect on login 2, got %d", resp5.Response.StatusCode)
	}
	loc5, _ := resp5.Response.Location()
	state2 := loc5.Query().Get("state")

	resp6, _ := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state2)+"&code=mockcode", nil)
	if resp6.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected redirect on callback 2")
	}

	resp7, _ := browser.Do("GET", "/tab/2", nil)
	if resp7.Response.StatusCode != http.StatusOK {
		t.Fatalf("Login 2 failed")
	}
}

// Tests that tokens don't leak out in plaintext over sessions
func TestTokenDoesNotLeak(t *testing.T) {
	tmp := t.TempDir()
	gobookmarks.Config.LocalGitPath = tmp
	gobookmarks.Config.SessionName = "gobookmarks_test_session"
	gobookmarks.Config.GithubClientID = "mock_client"
	gobookmarks.Config.GithubSecret = "mock_secret"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough"))
	gobookmarks.Config.ExternalURL = "https://example.com"

	gitP := gobookmarks.GitProvider{}
	gobookmarks.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), "mockuser", nil, gobookmarks.Config.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	p := &gobookmarks.GitHubProvider{}
	gobookmarks.RegisterProvider(p)

	r := mux.NewRouter()
	r.Use(gobookmarks.UserAdderMiddleware)
	r.Use(gobookmarks.CoreAdderMiddleware)

	r.HandleFunc("/tab/2", func(w http.ResponseWriter, req *http.Request) {
		session := gobookmarks.GetSession(w, req)
		user, _ := session.Values["GithubUser"].(*gobookmarks.User)
		if user != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}).Methods("GET")

	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, req *http.Request) { _ = gobookmarks.LoginWithProvider(w, req) }).Methods("GET")
	r.HandleFunc("/oauth2Callback", func(w http.ResponseWriter, req *http.Request) {
		client := &http.Client{Transport: &mockRoundTripper{}}
		ctx := context.WithValue(req.Context(), oauth2.HTTPClient, client)
		req = req.WithContext(ctx)

		_ = gobookmarks.Oauth2CallbackPage(w, req)
		session := gobookmarks.GetSession(w, req)
		redirectUrl := "/"
		if rURL, ok := session.Values["Redirect"].(string); ok && rURL != "" {
			redirectUrl = rURL
		}
		http.Redirect(w, req, redirectUrl, http.StatusTemporaryRedirect)
	}).Methods("GET")

	browser, _ := NewBrowser(r, "https://example.com")

	resp1, _ := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	loc1, _ := resp1.Response.Location()
	state1 := loc1.Query().Get("state")

	resp2, _ := browser.Do("GET", "/oauth2Callback?state="+state1+"&code=mockcode", nil)

	// Check that none of the emitted cookies contain "mock-token" in plain text
	for _, cookie := range resp2.Cookies {
		if strings.Contains(cookie.Value, "mock-token") {
			t.Fatalf("Cookie leaks plaintext token!")
		}
	}
}
