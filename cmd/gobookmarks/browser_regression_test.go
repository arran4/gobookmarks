package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gb "github.com/arran4/gobookmarks"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type mockOAuthRoundTripper struct {
	fakeToken string
	userLogin string
}

func (m *mockOAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	if strings.Contains(req.URL.Path, "access_token") {
		body := fmt.Sprintf(`{"access_token": %q, "token_type": "bearer"}`, m.fakeToken)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     header,
		}, nil
	}

	if strings.Contains(req.URL.Path, "user") {
		body := fmt.Sprintf(`{"login": %q}`, m.userLogin)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     header,
		}, nil
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("{}")),
		Header:     header,
	}, nil
}

func newTestProductionHarness(t *testing.T, fakeToken, userLogin string) (*Browser, *mux.Router) {
	t.Helper()
	tmpDir := t.TempDir()

	origConfig := gb.Config
	origStore := gb.SessionStore
	t.Cleanup(func() {
		gb.Config = origConfig
		gb.SessionStore = origStore
		gb.RegisterProvider(&gb.SQLProvider{})
	})

	gb.Config.LocalGitPath = tmpDir
	gb.Config.DBConnectionProvider = "sqlite3"
	gb.Config.DBConnectionString = filepath.Join(tmpDir, "test.db")
	gb.Config.SessionName = "gobookmarks_test_session"
	gb.Config.GithubClientID = "mock_client"
	gb.Config.GithubSecret = "mock_secret"
	gb.Config.ExternalURL = "https://example.com"
	gb.Config.ProviderOrder = []string{"github", "git", "sql"}
	gb.SetProviderOrder(gb.Config.ProviderOrder)
	gb.SessionStore = gb.InitSessionStore([]byte("regression-test-secret-key-that-is-long-enough-32bytes"))
	gb.SetVersion(version, commit, date)

	gitP := &gb.GitProvider{}
	gb.RegisterProvider(gitP)
	if err := gitP.CreateRepo(context.Background(), userLogin, nil, gb.Config.GetRepoName()); err != nil {
		t.Fatalf("git CreateRepo: %v", err)
	}

	sqlP := &gb.SQLProvider{}
	gb.RegisterProvider(sqlP)

	ghP := &gb.GitHubProvider{}
	gb.RegisterProvider(ghP)

	mockTransport := &mockOAuthRoundTripper{
		fakeToken: fakeToken,
		userLogin: userLogin,
	}
	mockClient := &http.Client{Transport: mockTransport}

	r := setupRouter()
	registerRoutes(r)

	// Wrap the actual production router with a test-only context injector
	// for oauth2.HTTPClient, avoiding process-global http.DefaultTransport mutation.
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(req.Context(), oauth2.HTTPClient, mockClient)
		r.ServeHTTP(w, req.WithContext(ctx))
	})

	browser, err := NewBrowser(testHandler, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}

	return browser, r
}

func TestBrowserSessionLifecycleRegression(t *testing.T) {
	fakeToken := "super-secret-unique-fake-oauth-token-xyz-12345"
	userLogin := "mockuser"
	browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

	// 1. Clean anonymous request: must not emit deletion cookies solely because session lacks version
	resp1, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatalf("GET /tab/2: %v", err)
	}
	if len(resp1.Cookies) > 0 {
		t.Fatalf("Anonymous request should not emit cookies merely due to missing version. Count: %d", len(resp1.Cookies))
	}

	// 2. Initiate Login with query-preserving redirect
	resp2, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2%3Fpage%3D3", nil)
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	if resp2.Response.StatusCode != http.StatusTemporaryRedirect && resp2.Response.StatusCode != http.StatusFound {
		t.Fatalf("Expected /login to redirect, got %d", resp2.Response.StatusCode)
	}
	loc2, err := resp2.Response.Location()
	if err != nil {
		t.Fatalf("Location header: %v", err)
	}
	stateParam := loc2.Query().Get("state")
	if stateParam == "" {
		t.Fatalf("Expected state parameter on OAuth redirect")
	}
	if !strings.HasPrefix(stateParam, "github:") {
		t.Fatalf("Expected state prefix github:, got: %s", stateParam)
	}
	if !strings.HasSuffix(stateParam, ":/tab/2?page=3") {
		t.Fatalf("Expected state suffix :/tab/2?page=3, got: %s", stateParam)
	}
	if len(resp2.Cookies) == 0 {
		t.Fatalf("Expected OAuth pending state cookie to be emitted")
	}

	// 3. Callback with valid state
	resp3, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(stateParam)+"&code=mockcode", nil)
	if err != nil {
		t.Fatalf("GET /oauth2Callback: %v", err)
	}
	if resp3.Response.StatusCode != http.StatusFound && resp3.Response.StatusCode != http.StatusTemporaryRedirect && resp3.Response.StatusCode != http.StatusSeeOther {
		t.Fatalf("Expected callback redirect, got %d", resp3.Response.StatusCode)
	}

	// Preserved redirect with query string
	loc3, err := resp3.Response.Location()
	if err != nil {
		t.Fatalf("Callback redirect Location header error: %v", err)
	}
	if loc3.Path != "/tab/2" || loc3.RawQuery != "page=3" {
		t.Fatalf("Expected post-login redirect to /tab/2?page=3, got path=%q rawQuery=%q", loc3.Path, loc3.RawQuery)
	}

	// Ordered Set-Cookie assertion
	mutations := AssertOrderedSetCookieMutations(t, resp3.Cookies)

	// Cookie Security, Size, and Token Confidentiality assertions
	sessionName := gb.Config.GetSessionName()
	var liveSessionCookie *http.Cookie
	for _, c := range resp3.Cookies {
		if strings.Contains(c.Value, fakeToken) {
			t.Fatalf("Token confidentiality regression: fake token plaintext leaked into cookie (name=%s, domain=%s, path=%s)", c.Name, c.Domain, c.Path)
		}
		if c.Name == sessionName {
			isLive := c.MaxAge > 0 || (c.MaxAge == 0 && (c.Expires.IsZero() || c.Expires.After(time.Now())))
			if isLive {
				liveSessionCookie = c
			}
		}
	}

	if liveSessionCookie == nil {
		t.Fatalf("No live session cookie found in callback response for %s. Mutations: %+v", sessionName, mutations)
	}

	if !liveSessionCookie.Secure {
		t.Fatalf("Session cookie missing Secure flag (name=%s, path=%s)", liveSessionCookie.Name, liveSessionCookie.Path)
	}
	if !liveSessionCookie.HttpOnly {
		t.Fatalf("Session cookie missing HttpOnly flag (name=%s, path=%s)", liveSessionCookie.Name, liveSessionCookie.Path)
	}
	if liveSessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("Session cookie SameSite = %v; want SameSiteLaxMode", liveSessionCookie.SameSite)
	}
	if liveSessionCookie.Path != "/" {
		t.Fatalf("Session cookie Path = %q; want /", liveSessionCookie.Path)
	}
	if len(liveSessionCookie.Value) > 4096 {
		t.Fatalf("Session cookie size %d exceeds 4096 bytes limit (name=%s, path=%s)", len(liveSessionCookie.Value), liveSessionCookie.Name, liveSessionCookie.Path)
	}

	// 4. Authenticated request
	resp4, err := browser.Do("GET", "/tab/2?page=3", nil)
	if err != nil {
		t.Fatalf("GET /tab/2?page=3: %v", err)
	}
	if resp4.Response.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK after login, got %d", resp4.Response.StatusCode)
	}
	body4, _ := io.ReadAll(resp4.Response.Body)
	if !bytes.Contains(body4, []byte("Logout")) {
		t.Fatalf("Expected authenticated page to contain Logout button")
	}

	// 5. Repeat Login: Authenticated user initiating login again
	resp5, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	if err != nil {
		t.Fatalf("Repeat GET /login: %v", err)
	}
	loc5, _ := resp5.Response.Location()
	state5 := loc5.Query().Get("state")

	resp6, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state5)+"&code=mockcode2", nil)
	if err != nil {
		t.Fatalf("Repeat callback: %v", err)
	}
	if resp6.Response.StatusCode != http.StatusFound && resp6.Response.StatusCode != http.StatusSeeOther && resp6.Response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Expected repeat callback redirect, got %d", resp6.Response.StatusCode)
	}
	AssertOrderedSetCookieMutations(t, resp6.Cookies)

	resp7, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatalf("GET /tab/2 after repeat login: %v", err)
	}
	body7, _ := io.ReadAll(resp7.Response.Body)
	if !bytes.Contains(body7, []byte("Logout")) {
		t.Fatalf("Expected authenticated page after repeat login to contain Logout button")
	}

	// 6. Logout
	resp8, err := browser.Do("GET", "/logout", nil)
	if err != nil {
		t.Fatalf("GET /logout: %v", err)
	}
	if resp8.Response.StatusCode != http.StatusOK {
		t.Fatalf("Expected /logout 200 OK, got %d", resp8.Response.StatusCode)
	}

	// Assert logout emits deliberate/singular invalidation rather than contradictory mutations
	logoutMutations := AssertOrderedSetCookieMutations(t, resp8.Cookies)
	foundLogoutExpiry := false
	for id, history := range logoutMutations {
		if id.Name == sessionName {
			if len(history) != 1 {
				t.Fatalf("Expected singular Set-Cookie mutation on logout for session cookie, got %d mutations: %+v", len(history), history)
			}
			if !history[0].IsExpiry {
				t.Fatalf("Expected logout mutation to be an expiry, got: %+v", history[0])
			}
			foundLogoutExpiry = true
		}
	}
	if !foundLogoutExpiry {
		t.Fatalf("Expected logout to emit an invalidation Set-Cookie for %s", sessionName)
	}

	// 7. Subsequent request after logout is unauthenticated
	resp9, err := browser.Do("GET", "/tab/2", nil)
	if err != nil {
		t.Fatalf("GET /tab/2 after logout: %v", err)
	}
	body9, _ := io.ReadAll(resp9.Response.Body)
	if bytes.Contains(body9, []byte("Logout")) {
		t.Fatalf("Expected unauthenticated page after logout not to contain Logout button")
	}
}

func TestBrowserOAuthStateNegativeMatrix(t *testing.T) {
	fakeToken := "oauth-matrix-fake-token"
	userLogin := "matrixuser"

	t.Run("MissingState", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		// Initiate login to get OAuth pending session with nonce
		respLogin, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(respLogin.Cookies) == 0 {
			t.Fatalf("Expected pending cookie")
		}

		// Callback missing state
		respCb, err := browser.Do("GET", "/oauth2Callback?code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respCb.Response.StatusCode == http.StatusFound || respCb.Response.StatusCode == http.StatusSeeOther || respCb.Response.StatusCode == http.StatusTemporaryRedirect {
			loc, _ := respCb.Response.Location()
			if loc != nil && loc.Path == "/tab/2" {
				t.Fatalf("Callback should have failed on missing state, but redirected to /tab/2")
			}
		}

		respTab, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(respTab.Response.Body)
		if bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected user to remain unauthenticated on missing state")
		}
	})

	t.Run("MalformedState", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		_, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
		if err != nil {
			t.Fatal(err)
		}

		malformedStates := []string{
			"malformed",
			"github",
			"github:",
			"otherprovider:nonce:/tab/2",
		}

		for _, badState := range malformedStates {
			respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(badState)+"&code=mockcode", nil)
			if err != nil {
				t.Fatal(err)
			}
			if respCb.Response.StatusCode == http.StatusFound || respCb.Response.StatusCode == http.StatusSeeOther {
				loc, _ := respCb.Response.Location()
				if loc != nil && loc.Path == "/tab/2" {
					t.Fatalf("Callback succeeded unexpectedly with malformed state %q", badState)
				}
			}

			respTab, err := browser.Do("GET", "/tab/2", nil)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(respTab.Response.Body)
			if bytes.Contains(body, []byte("Logout")) {
				t.Fatalf("Expected user to remain unauthenticated with malformed state %q", badState)
			}
		}
	})

	t.Run("WrongNonce", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		_, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
		if err != nil {
			t.Fatal(err)
		}

		wrongState := "github:0123456789abcdef0123456789abcdef:/tab/2"
		respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(wrongState)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respCb.Response.StatusCode == http.StatusFound || respCb.Response.StatusCode == http.StatusSeeOther {
			loc, _ := respCb.Response.Location()
			if loc != nil && loc.Path == "/tab/2" {
				t.Fatalf("Callback succeeded unexpectedly with wrong nonce")
			}
		}

		respTab, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(respTab.Response.Body)
		if bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected user to remain unauthenticated with wrong nonce")
		}
	})

	t.Run("ValidNonceAndReplayConsumption", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		respLogin, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2", nil)
		if err != nil {
			t.Fatal(err)
		}
		loc, _ := respLogin.Response.Location()
		validState := loc.Query().Get("state")

		// 1. First callback with valid state succeeds
		respCb1, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(validState)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respCb1.Response.StatusCode != http.StatusFound && respCb1.Response.StatusCode != http.StatusSeeOther {
			t.Fatalf("Expected successful callback redirect, got %d", respCb1.Response.StatusCode)
		}

		respTab1, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		body1, _ := io.ReadAll(respTab1.Response.Body)
		if !bytes.Contains(body1, []byte("Logout")) {
			t.Fatalf("Expected user to be authenticated after valid callback")
		}

		// 2. Replay the exact same callback URL: must fail because nonce was consumed
		respCb2, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(validState)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respCb2.Response.StatusCode == http.StatusFound || respCb2.Response.StatusCode == http.StatusSeeOther {
			loc2, _ := respCb2.Response.Location()
			if loc2 != nil && loc2.Path == "/tab/2" {
				t.Fatalf("Nonce replay attack succeeded: consumed nonce was re-accepted")
			}
		}
	})
}

func TestBrowserRedirectPreservationAndSafety(t *testing.T) {
	fakeToken := "redirect-safety-token"
	userLogin := "redirectuser"

	t.Run("PreservesPathAndQuery", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		respLogin, err := browser.Do("GET", "/login/github?redirect=%2Ftab%2F2%3Fpage%3D3", nil)
		if err != nil {
			t.Fatal(err)
		}
		loc, _ := respLogin.Response.Location()
		state := loc.Query().Get("state")

		respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		cbLoc, err := respCb.Response.Location()
		if err != nil {
			t.Fatalf("Callback redirect location error: %v", err)
		}
		if cbLoc.Path != "/tab/2" || cbLoc.RawQuery != "page=3" {
			t.Fatalf("Expected redirect to preserve /tab/2?page=3, got path=%q query=%q", cbLoc.Path, cbLoc.RawQuery)
		}
	})

	t.Run("PreventsOpenRedirectViaScheme", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		respLogin, err := browser.Do("GET", "/login/github?redirect=https%3A%2F%2Fevil.com%2Fphish", nil)
		if err != nil {
			t.Fatal(err)
		}
		loc, _ := respLogin.Response.Location()
		state := loc.Query().Get("state")

		respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		cbLoc, err := respCb.Response.Location()
		if err != nil {
			t.Fatalf("Callback redirect location error: %v", err)
		}
		if cbLoc.Host != "" || cbLoc.Scheme != "" || cbLoc.Path != "/" {
			t.Fatalf("Expected safe fallback to /, got: %s", cbLoc.String())
		}
	})

	t.Run("PreventsProtocolRelativeRedirect", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		respLogin, err := browser.Do("GET", "/login/github?redirect=%2F%2Fevil.com%2Fphish", nil)
		if err != nil {
			t.Fatal(err)
		}
		loc, _ := respLogin.Response.Location()
		state := loc.Query().Get("state")

		respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		cbLoc, err := respCb.Response.Location()
		if err != nil {
			t.Fatalf("Callback redirect location error: %v", err)
		}
		if cbLoc.Host != "" || cbLoc.Scheme != "" || cbLoc.Path != "/" {
			t.Fatalf("Expected safe fallback to /, got: %s", cbLoc.String())
		}
	})

	t.Run("PreventsBackslashRedirect", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		respLogin, err := browser.Do("GET", "/login/github?redirect=%2F%5Cevil.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		loc, _ := respLogin.Response.Location()
		state := loc.Query().Get("state")

		respCb, err := browser.Do("GET", "/oauth2Callback?state="+url.QueryEscape(state)+"&code=mockcode", nil)
		if err != nil {
			t.Fatal(err)
		}
		cbLoc, err := respCb.Response.Location()
		if err != nil {
			t.Fatalf("Callback redirect location error: %v", err)
		}
		if cbLoc.Host != "" || cbLoc.Scheme != "" || cbLoc.Path != "/" {
			t.Fatalf("Expected safe fallback to /, got: %s", cbLoc.String())
		}
	})
}

func TestBrowserSessionLifecycleMatrix(t *testing.T) {
	fakeToken := "session-matrix-token"
	userLogin := "lifecycleuser"

	t.Run("CleanAnonymous", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)
		resp, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Cookies) > 0 {
			t.Fatalf("Expected 0 cookies on clean anonymous request, got %d", len(resp.Cookies))
		}
	})

	t.Run("OAuthPendingSessionWithoutVersion", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)
		resp, err := browser.Do("GET", "/login/github?redirect=/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Cookies) == 0 {
			t.Fatalf("Expected pending session cookie")
		}

		// Subsequent request before auth should not invalidate the pending state
		resp2, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		// Confirm it didn't emit deletion cookie
		for _, c := range resp2.Cookies {
			if c.Name == gb.Config.GetSessionName() && c.MaxAge < 0 {
				t.Fatalf("OAuth pending session unexpectedly invalidated with deletion cookie")
			}
		}
	})

	t.Run("OldVersionAuthenticatedSessionIsInvalidated", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		// Create a session cookie with an old version
		targetURL, _ := browser.Origin.Parse("/tab/2")
		mockReq := httptest.NewRequest("GET", "https://example.com/tab/2", nil)
		mockRec := httptest.NewRecorder()

		sess, _ := gb.SessionStore.New(mockReq, gb.Config.GetSessionName())
		sess.Values["GithubUser"] = &gb.User{Login: userLogin}
		sess.Values["version"] = "old-version-1.0"
		_ = sess.Save(mockReq, mockRec)

		browser.Jar.SetCookies(targetURL, mockRec.Result().Cookies())

		resp, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Response.Body)
		if bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected old version session to be invalidated, but page shows authenticated")
		}

		// Check that an invalidation mutation was emitted
		foundExpiry := false
		for _, c := range resp.Cookies {
			if c.Name == gb.Config.GetSessionName() && (c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(time.Now()))) {
				foundExpiry = true
			}
		}
		if !foundExpiry {
			t.Fatalf("Expected deletion cookie for old version session")
		}
	})

	t.Run("LegacyAuthenticatedSessionLackingVersionIsInvalidated", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		targetURL, _ := browser.Origin.Parse("/tab/2")
		mockReq := httptest.NewRequest("GET", "https://example.com/tab/2", nil)
		mockRec := httptest.NewRecorder()

		sess, _ := gb.SessionStore.New(mockReq, gb.Config.GetSessionName())
		sess.Values["GithubUser"] = &gb.User{Login: userLogin}
		// Omit version completely
		_ = sess.Save(mockReq, mockRec)

		browser.Jar.SetCookies(targetURL, mockRec.Result().Cookies())

		resp, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Response.Body)
		if bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected legacy session lacking version to be invalidated, but page shows authenticated")
		}

		foundExpiry := false
		for _, c := range resp.Cookies {
			if c.Name == gb.Config.GetSessionName() && (c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(time.Now()))) {
				foundExpiry = true
			}
		}
		if !foundExpiry {
			t.Fatalf("Expected deletion cookie for legacy session lacking version")
		}
	})

	t.Run("MalformedCookieRecoversCleanly", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		targetURL, _ := browser.Origin.Parse("/tab/2")
		browser.Jar.SetCookies(targetURL, []*http.Cookie{
			{
				Name:  gb.Config.GetSessionName(),
				Value: "tampered-data-invalid-mac-or-decode",
				Path:  "/",
			},
		})

		resp, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Response.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK clean recovery from bad cookie, got %d", resp.Response.StatusCode)
		}

		// Ensure bad cookie is wiped from jar
		for _, c := range browser.Jar.Cookies(targetURL) {
			if c.Name == gb.Config.GetSessionName() && c.Value == "tampered-data-invalid-mac-or-decode" {
				t.Fatalf("Malformed cookie was not properly cleared from jar")
			}
		}
	})
}

func TestBrowserGitAndSqlAuthFlows(t *testing.T) {
	fakeToken := "git-sql-token"
	userLogin := "testuser"

	t.Run("GitLoginAndLogoutFlow", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		// Create user in Git provider
		gitP := gb.GetProvider("git")
		ph, ok := gitP.(gb.PasswordHandler)
		if !ok {
			t.Fatalf("Git provider does not implement PasswordHandler")
		}
		if err := ph.CreateUser(context.Background(), "gituser", "correctpass"); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if err := gitP.CreateRepo(context.Background(), "gituser", nil, gb.Config.GetRepoName()); err != nil {
			t.Fatalf("CreateRepo: %v", err)
		}
		if err := gitP.CreateBookmarks(context.Background(), "gituser", nil, "main", "initial bookmarks"); err != nil {
			t.Fatalf("CreateBookmarks: %v", err)
		}

		// Login via POST /login/git with redirect
		form := url.Values{
			"username": {"gituser"},
			"password": {"correctpass"},
			"redirect": {"/tab/2?page=3"},
		}
		respLogin, err := browser.DoForm("/login/git", form)
		if err != nil {
			t.Fatalf("POST /login/git: %v", err)
		}
		if respLogin.Response.StatusCode != http.StatusSeeOther && respLogin.Response.StatusCode != http.StatusFound {
			t.Fatalf("Expected login redirect, got %d", respLogin.Response.StatusCode)
		}
		loc, _ := respLogin.Response.Location()
		if loc.Path != "/tab/2" || loc.RawQuery != "page=3" {
			t.Fatalf("Expected redirect to /tab/2?page=3, got %s", loc.String())
		}

		// Verify no live-then-expiry regression on login
		AssertOrderedSetCookieMutations(t, respLogin.Cookies)

		// Check authenticated access
		respTab, err := browser.Do("GET", "/tab/2?page=3", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respTab.Response.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK on authenticated tab, got %d", respTab.Response.StatusCode)
		}
		body, _ := io.ReadAll(respTab.Response.Body)
		if !bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected authenticated page to contain Logout button")
		}

		// Logout
		respLogout, err := browser.Do("GET", "/logout", nil)
		if err != nil {
			t.Fatal(err)
		}
		AssertOrderedSetCookieMutations(t, respLogout.Cookies)

		// Post-logout check
		respPost, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		bodyPost, _ := io.ReadAll(respPost.Response.Body)
		if bytes.Contains(bodyPost, []byte("Logout")) {
			t.Fatalf("Expected unauthenticated page after logout")
		}
	})

	t.Run("SqlLoginAndLogoutFlow", func(t *testing.T) {
		browser, _ := newTestProductionHarness(t, fakeToken, userLogin)

		sqlP := gb.GetProvider("sql")
		ph, ok := sqlP.(gb.PasswordHandler)
		if !ok {
			t.Fatalf("SQL provider does not implement PasswordHandler")
		}
		if err := ph.CreateUser(context.Background(), "sqluser", "correctpass"); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if err := sqlP.CreateRepo(context.Background(), "sqluser", nil, gb.Config.GetRepoName()); err != nil {
			t.Fatalf("CreateRepo: %v", err)
		}
		if err := sqlP.CreateBookmarks(context.Background(), "sqluser", nil, "main", "initial bookmarks"); err != nil {
			t.Fatalf("CreateBookmarks: %v", err)
		}

		form := url.Values{
			"username": {"sqluser"},
			"password": {"correctpass"},
			"redirect": {"/tab/2?page=3"},
		}
		respLogin, err := browser.DoForm("/login/sql", form)
		if err != nil {
			t.Fatalf("POST /login/sql: %v", err)
		}
		if respLogin.Response.StatusCode != http.StatusSeeOther && respLogin.Response.StatusCode != http.StatusFound {
			t.Fatalf("Expected login redirect, got %d", respLogin.Response.StatusCode)
		}
		loc, _ := respLogin.Response.Location()
		if loc.Path != "/tab/2" || loc.RawQuery != "page=3" {
			t.Fatalf("Expected redirect to /tab/2?page=3, got %s", loc.String())
		}

		AssertOrderedSetCookieMutations(t, respLogin.Cookies)

		respTab, err := browser.Do("GET", "/tab/2?page=3", nil)
		if err != nil {
			t.Fatal(err)
		}
		if respTab.Response.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK on authenticated tab, got %d", respTab.Response.StatusCode)
		}
		body, _ := io.ReadAll(respTab.Response.Body)
		if !bytes.Contains(body, []byte("Logout")) {
			t.Fatalf("Expected authenticated page to contain Logout button")
		}

		respLogout, err := browser.Do("GET", "/logout", nil)
		if err != nil {
			t.Fatal(err)
		}
		AssertOrderedSetCookieMutations(t, respLogout.Cookies)

		respPost, err := browser.Do("GET", "/tab/2", nil)
		if err != nil {
			t.Fatal(err)
		}
		bodyPost, _ := io.ReadAll(respPost.Response.Body)
		if bytes.Contains(bodyPost, []byte("Logout")) {
			t.Fatalf("Expected unauthenticated page after logout")
		}
	})
}
