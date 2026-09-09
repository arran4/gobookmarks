package gobookmarks

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLoginRouteProviderVariable(t *testing.T) {
	Config.SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"
	Config.GithubClientID = "id"
	Config.GithubSecret = "secret"
	Config.GitlabClientID = "id"
	Config.GitlabSecret = "secret"
	Config.ExternalURL = "http://example.com/"

	r := mux.NewRouter()
	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, r *http.Request) { _ = LoginWithProvider(w, r) }).Methods("GET")

	req := httptest.NewRequest("GET", "/login/github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect, got %d", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.Contains(loc, "github") {
		t.Fatalf("redirect location does not contain provider: %s", loc)
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse OAuth redirect: %v", err)
	}
	if state := parsed.Query().Get("state"); state != "github" {
		t.Fatalf("default redirect must not be included in OAuth state, got %q", state)
	}

	req = httptest.NewRequest("GET", "/login/github?redirect=%2F", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	parsed, err = url.Parse(w.Result().Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse OAuth redirect with default return URL: %v", err)
	}
	if state := parsed.Query().Get("state"); state != "github" {
		t.Fatalf("default redirect must not be included in OAuth state, got %q", state)
	}

	req = httptest.NewRequest("GET", "/login/unknown", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown provider, got %d", w.Result().StatusCode)
	}
}

func TestLoginRouteProviderVariableRedirect(t *testing.T) {
	Config.SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"
	Config.GithubClientID = "id"
	Config.GithubSecret = "secret"
	Config.GitlabClientID = "id"
	Config.GitlabSecret = "secret"
	Config.ExternalURL = "http://example.com/"

	r := mux.NewRouter()
	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, r *http.Request) { _ = LoginWithProvider(w, r) }).Methods("GET")

	req := httptest.NewRequest("GET", "/login/github?redirect=%2Ftab%2F2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	parsed, err := url.Parse(w.Result().Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse OAuth redirect with explicit return URL: %v", err)
	}
	if state := parsed.Query().Get("state"); state != "github:/tab/2" {
		t.Fatalf("explicit redirect must be included in OAuth state, got %q", state)
	}

	req = httptest.NewRequest("GET", "/login/github?redirect=%2Ftab%2F2%3Fpage%3D3", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	parsed, err = url.Parse(w.Result().Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse OAuth redirect with explicit complex return URL: %v", err)
	}
	if state := parsed.Query().Get("state"); state != "github:/tab/2?page=3" {
		t.Fatalf("explicit complex redirect must be included in OAuth state, got %q", state)
	}
}

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// If it's a token exchange, return a dummy token
	if strings.Contains(req.URL.String(), "access_token") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"mocktoken","token_type":"bearer"}`)),
		}, nil
	}
	// If it's fetching the user, return a dummy user
	if strings.Contains(req.URL.String(), "user") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"login":"mockuser"}`)),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestOauth2CallbackRedirect(t *testing.T) {
	Config.SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"
	Config.GithubClientID = "id"
	Config.GithubSecret = "secret"
	Config.GitlabClientID = "id"
	Config.GitlabSecret = "secret"
	Config.ExternalURL = "http://example.com/"

	// Set up the mock client
	client := &http.Client{Transport: &mockRoundTripper{}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	// Test the Oauth2CallbackPage to ensure it extracts the redirect properly from the state parameter
	req := httptest.NewRequest("GET", "/oauth2Callback?state=github:/tab/2?page=3&code=mockcode", nil)
	w := httptest.NewRecorder()

	session, _ := SessionStore.Get(req, Config.GetSessionName())
	session.Values["version"] = version
	ctx = context.WithValue(ctx, ContextValues("session"), session)
	req = req.WithContext(ctx)

	// Since we mocked the token exchange and user fetching, Oauth2CallbackPage should succeed and return nil
	err := Oauth2CallbackPage(w, req)
	if err != nil {
		t.Fatalf("Oauth2CallbackPage failed: %v", err)
	}

	// Read the new session from the response cookies
	req2_check := httptest.NewRequest("GET", "/", nil)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("No cookies returned from Oauth2CallbackPage!")
	}
	// Add only the valid cookie (the last one or the one with Max-Age > 0)
	for _, cookie := range cookies {
		if cookie.MaxAge > 0 {
			req2_check.AddCookie(cookie)
		}
	}
	session2, err2 := SessionStore.Get(req2_check, Config.GetSessionName())
	if err2 != nil {
		t.Logf("SessionStore.Get error: %v", err2)
	}

	if session2.Values["Redirect"] != "/tab/2?page=3" {
		t.Fatalf("Expected Oauth2CallbackPage to set session Redirect to /tab/2?page=3, got: %v (All values: %v)", session2.Values["Redirect"], session2.Values)
	}

}
