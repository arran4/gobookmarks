package gobookmarks

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRouteProviderVariable(t *testing.T) {
	config := NewConfiguration()
	config.SessionName = "testsess"
	config.SessionStore = sessions.NewCookieStore([]byte("secret"))
	version = "vtest"
	config.GithubClientID = "id"
	config.GithubSecret = "secret"
	config.GitlabClientID = "id"
	config.GitlabSecret = "secret"
	config.OauthRedirectURL = JoinURL("http://example.com/", "callback")

	r := mux.NewRouter()
	r.HandleFunc("/login/{provider}", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ContextValues("configuration"), config)
		r = r.WithContext(ctx)
		_ = LoginWithProvider(w, r)
	}).Methods("GET")

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
