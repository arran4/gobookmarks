package main

import (
	"context"
	"github.com/arran4/gobookmarks"

	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "access_token") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"mocktoken","token_type":"bearer"}`)),
		}, nil
	}
	if strings.Contains(req.URL.String(), "user") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"login":"mockuser"}`)),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestRedirectToHandlerIntegration(t *testing.T) {
	// Initialize minimal configuration
	gobookmarks.Config.SessionName = "testsess"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("secret"))

	// 1. Simulate setting up a successful session with Redirect set
	req := httptest.NewRequest("GET", "/login", nil)
	session := gobookmarks.GetSession(nil, req)
	session.Values["Redirect"] = "/tab/2?page=3"

	w := httptest.NewRecorder()
	_ = session.Save(req, w)

	// We create a new request simulating the next step in the chain where `redirectToHandler("/")` is executed
	req2 := httptest.NewRequest("GET", "/some_callback_path", nil)
	// Add the session cookie so redirectToHandler finds it
	for _, cookie := range w.Result().Cookies() {
		if cookie.MaxAge > 0 {
			req2.AddCookie(cookie)
		}
	}

	w2 := httptest.NewRecorder()

	// Execute redirectToHandler directly
	handler := redirectToHandler("/")
	handler(w2, req2)

	loc := w2.Result().Header.Get("Location")
	if loc != "/tab/2?page=3" {
		t.Fatalf("Expected redirectToHandler to consume session Redirect and point to /tab/2?page=3, got: %s", loc)
	}
}

func TestFullLoginChainIntegration(t *testing.T) {
	gobookmarks.Config.SessionName = "testsess"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte("secret"))

	d := t.TempDir()
	gobookmarks.Config.LocalGitPath = d
	gobookmarks.Config.DBConnectionProvider = "sqlite3"
	gobookmarks.Config.DBConnectionString = d + "/test.db"
	gobookmarks.RegisterProvider(&gobookmarks.GitProvider{})
	gobookmarks.RegisterProvider(&gobookmarks.SQLProvider{})

	// 1. Simulate a successful Git signup
	req := httptest.NewRequest("POST", "/signup/git", strings.NewReader("username=bob&password=secret&redirect=/tab/2?page=3"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	session := gobookmarks.GetSession(nil, req)
	gobookmarks.SetVersion("vtest", "c", "d")
	session.Values["version"] = "vtest"
	ctx := context.WithValue(req.Context(), gobookmarks.ContextValues("session"), session)
	req = req.WithContext(ctx)

	handlerSignupGit := runHandlerChain(gobookmarks.GitSignupAction, redirectToHandler("/login/git"))
	handlerSignupGit(w, req)

	loc := w.Result().Header.Get("Location")
	if loc != "/login/git?redirect=%2Ftab%2F2%3Fpage%3D3" {
		t.Fatalf("Expected git signup to redirect to /login/git?redirect=%%2Ftab%%2F2%%3Fpage%%3D3, got: %s", loc)
	}

	// 2. Simulate a successful Git login via chain
	req2 := httptest.NewRequest("POST", "/login/git", strings.NewReader("username=bob&password=secret&redirect=/tab/2?page=3"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w2 := httptest.NewRecorder()

	for _, cookie := range w.Result().Cookies() {
		if cookie.MaxAge > 0 {
			req2.AddCookie(cookie)
		}
	}
	session2 := gobookmarks.GetSession(nil, req2)
	session2.Values["version"] = "vtest"
	ctx2 := context.WithValue(req2.Context(), gobookmarks.ContextValues("session"), session2)
	req2 = req2.WithContext(ctx2)

	handlerLoginGit := runHandlerChain(gobookmarks.GitLoginAction, redirectToHandler("/"))
	handlerLoginGit(w2, req2)

	loc2 := w2.Result().Header.Get("Location")
	if loc2 != "/tab/2?page=3" {
		t.Fatalf("Expected git login to eventually redirect to /tab/2?page=3 via redirectToHandler, got: %s", loc2)
	}

	// 3. Simulate a successful SQL signup
	req3 := httptest.NewRequest("POST", "/signup/sql", strings.NewReader("username=sue&password=secret&redirect=/tab/2?page=3"))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w3 := httptest.NewRecorder()

	session3 := gobookmarks.GetSession(nil, req3)
	session3.Values["version"] = "vtest"
	ctx3 := context.WithValue(req3.Context(), gobookmarks.ContextValues("session"), session3)
	req3 = req3.WithContext(ctx3)

	handlerSignupSql := runHandlerChain(gobookmarks.SqlSignupAction, redirectToHandler("/login/sql"))
	handlerSignupSql(w3, req3)

	loc3 := w3.Result().Header.Get("Location")
	if loc3 != "/login/sql?redirect=%2Ftab%2F2%3Fpage%3D3" {
		t.Fatalf("Expected sql signup to redirect to /login/sql?redirect=%%2Ftab%%2F2%%3Fpage%%3D3, got: %s", loc3)
	}

	// 4. Simulate a successful SQL login via chain
	req4 := httptest.NewRequest("POST", "/login/sql", strings.NewReader("username=sue&password=secret&redirect=/tab/2?page=3"))
	req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w4 := httptest.NewRecorder()

	for _, cookie := range w3.Result().Cookies() {
		if cookie.MaxAge > 0 {
			req4.AddCookie(cookie)
		}
	}
	session4 := gobookmarks.GetSession(nil, req4)
	session4.Values["version"] = "vtest"
	ctx4 := context.WithValue(req4.Context(), gobookmarks.ContextValues("session"), session4)
	req4 = req4.WithContext(ctx4)

	handlerLoginSql := runHandlerChain(gobookmarks.SqlLoginAction, redirectToHandler("/"))
	handlerLoginSql(w4, req4)

	loc4 := w4.Result().Header.Get("Location")
	if loc4 != "/tab/2?page=3" {
		t.Fatalf("Expected sql login to eventually redirect to /tab/2?page=3 via redirectToHandler, got: %s", loc4)
	}

	// 5. Simulate Oauth callback success via chain
	req5 := httptest.NewRequest("GET", "/oauth2Callback?state=github:mocknonce:/tab/2?page=3&code=mockcode", nil)
	w5 := httptest.NewRecorder()

	session5 := gobookmarks.GetSession(nil, req5)
	session5.Values["version"] = "vtest"
	session5.Values["OauthState"] = "mocknonce"

	client := &http.Client{Transport: &mockRoundTripper{}}
	ctx5 := context.WithValue(req5.Context(), oauth2.HTTPClient, client)
	ctx5 = context.WithValue(ctx5, gobookmarks.ContextValues("session"), session5)
	req5 = req5.WithContext(ctx5)

	gobookmarks.Config.GithubClientID = "id"
	gobookmarks.Config.GithubSecret = "secret"
	gobookmarks.RegisterProvider(&gobookmarks.GitHubProvider{})

	handlerOauth := runHandlerChain(gobookmarks.Oauth2CallbackPage, redirectToHandler("/"))
	handlerOauth(w5, req5)

	loc5 := w5.Result().Header.Get("Location")
	if loc5 != "/tab/2?page=3" {
		t.Fatalf("Expected oauth callback to eventually redirect to /tab/2?page=3 via redirectToHandler, got: %s", loc5)
	}
}

// We verify the compatibility route caching headers directly using an HTTP test rather than trying
// to start the full `serve` command.
func TestCompatibilityRouteCacheHeaders(t *testing.T) {
	// setupRouter builds the production router exactly as the serve command uses it
	r := setupRouter()

	// Test /main.css
	reqCSS := httptest.NewRequest("GET", "/main.css", nil)
	recCSS := httptest.NewRecorder()
	r.ServeHTTP(recCSS, reqCSS)

	if recCSS.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /main.css, got %q", recCSS.Header().Get("Cache-Control"))
	}

	if recCSS.Code != http.StatusFound && recCSS.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /main.css, got %d", recCSS.Code)
	}

	// Test /favicon.ico
	reqFavicon := httptest.NewRequest("GET", "/favicon.ico", nil)
	recFavicon := httptest.NewRecorder()
	r.ServeHTTP(recFavicon, reqFavicon)

	if recFavicon.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /favicon.ico, got %q", recFavicon.Header().Get("Cache-Control"))
	}

	if recFavicon.Code != http.StatusFound && recFavicon.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /favicon.ico, got %d", recFavicon.Code)
	}
}
