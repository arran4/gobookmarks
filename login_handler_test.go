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
