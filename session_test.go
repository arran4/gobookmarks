package gobookmarks

import (
	"github.com/gorilla/sessions"
	"net/http/httptest"
	"testing"
)

// Test that getSession clears outdated sessions and returns a fresh one.
func Test_getSessionClearsOldVersion(t *testing.T) {
	SessionName = "testsession"
	SessionStore = sessions.NewCookieStore([]byte("secret-key"))
	version = "current"

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// create a session with an old version
	s, _ := SessionStore.New(req, SessionName)
	s.Values["version"] = "old"
	s.Values["GithubUser"] = &User{Login: "old"}
	if err := s.Save(req, w); err != nil {
		t.Fatalf("save old session: %v", err)
	}
	cookie := w.Result().Cookies()[0]
	req.AddCookie(cookie)

	w = httptest.NewRecorder()
	newSession, err := getSession(w, req)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	if len(newSession.Values) != 0 {
		t.Fatalf("expected empty session, got %#v", newSession.Values)
	}
	if h := w.Header().Get("Set-Cookie"); h == "" {
		t.Fatalf("expected Set-Cookie header to clear session")
	}
}
