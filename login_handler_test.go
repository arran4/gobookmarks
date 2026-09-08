package gobookmarks

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
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
	r.HandleFunc("/login/{provider
	// Test the Oauth2CallbackPage to ensure it extracts the redirect properly from the state parameter
	req = httptest.NewRequest("GET", "/oauth2Callback?state=github:/tab/2?page=3", nil)
	w = httptest.NewRecorder()

	session, _ := SessionStore.Get(req, Config.GetSessionName())
	session.Values["version"] = version
	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	req = req.WithContext(ctx)

	// Note: Oauth2CallbackPage actually exchanges the token. Without a real server, it will fail: "exchange error".
	// But it sets session.Values["Redirect"] BEFORE failing!
	_ = Oauth2CallbackPage(w, req)

	if session.Values["Redirect"] != "/tab/2?page=3" {
		t.Fatalf("Expected Oauth2CallbackPage to set session Redirect to /tab/2?page=3, got: %v", session.Values["Redirect"])
	}
}", func(w http.ResponseWriter, r *http.Request) { _ = LoginWithProvider(w, r) }).Methods("GET")

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

func TestOauth2CallbackRedirect(t *testing.T) {
	Config.SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"
	Config.GithubClientID = "id"
	Config.GithubSecret = "secret"
	Config.GitlabClientID = "id"
	Config.GitlabSecret = "secret"
	Config.ExternalURL = "http://example.com/"

	// Test the Oauth2CallbackPage to ensure it extracts the redirect properly from the state parameter
	req := httptest.NewRequest("GET", "/oauth2Callback?state=github:/tab/2?page=3", nil)
	w := httptest.NewRecorder()

	session, _ := SessionStore.Get(req, Config.GetSessionName())
	session.Values["version"] = version
	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	req = req.WithContext(ctx)

	// Note: Oauth2CallbackPage actually exchanges the token. Without a real server, it will fail: "exchange error".
	// But it sets session.Values["Redirect"] BEFORE failing!
	_ = Oauth2CallbackPage(w, req)

	if session.Values["Redirect"] != "/tab/2?page=3" {
		t.Fatalf("Expected Oauth2CallbackPage to set session Redirect to /tab/2?page=3, got: %v", session.Values["Redirect"])
	}
}
