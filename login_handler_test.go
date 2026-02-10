package gobookmarks

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRouteProviderVariable(t *testing.T) {
	cfg := NewConfiguration(Config{
		GithubClientID: "id",
		GithubSecret:   "secret",
		GitlabClientID: "id",
		GitlabSecret:   "secret",
		ExternalURL:    "http://example.com/",
	})
	cfg.SessionName = "testsess"
	cfg.SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"

	r := mux.NewRouter()
	r.Use(ConfigMiddleware(cfg))
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

	req = httptest.NewRequest("GET", "/login/unknown", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown provider, got %d", w.Result().StatusCode)
	}
}
