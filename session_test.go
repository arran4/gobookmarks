package gobookmarks

import (
	"net/http/httptest"
	"testing"
)

// Test that GetSession clears outdated sessions and returns a fresh one.
func Test_GetSessionClearsOldVersion(t *testing.T) {
	Config.SessionName = "testsession"
	SessionStore = InitSessionStore([]byte("secret-key"))
	version = "current"

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// create a session with an old version
	s, _ := SessionStore.New(req, Config.SessionName)
	s.Values["version"] = "old"
	s.Values["GithubUser"] = &User{Login: "old"}
	if err := s.Save(req, w); err != nil {
		t.Fatalf("save old session: %v", err)
	}
	cookie := w.Result().Cookies()[0]
	req.AddCookie(cookie)

	w = httptest.NewRecorder()
	newSession := GetSession(w, req)
	if newSession == nil {
		t.Fatalf("GetSession failed")
	}
	if len(newSession.Values) != 0 {
		t.Fatalf("expected empty session, got %#v", newSession.Values)
	}
	if h := w.Header().Get("Set-Cookie"); h == "" {
		t.Fatalf("expected Set-Cookie header to clear session")
	}
}
