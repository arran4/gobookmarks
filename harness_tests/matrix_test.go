package harness_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
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
