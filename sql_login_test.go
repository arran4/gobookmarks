package gobookmarks

import (
		"github.com/gorilla/sessions"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestSqlSignupScenarioWithRedirect(t *testing.T) {
	d := t.TempDir()

	// Create an empty db file to prevent errors
	dbFile := d + "/test.db"
	f, err := os.Create(dbFile)
	if err == nil {
		_ = f.Close()
	}

	Config.DBConnectionProvider = "sqlite3"
	Config.DBConnectionString = dbFile

	Config.SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))

	// signup
	form := url.Values{"username": []string{"bob"}, "password": []string{"secret"}, "redirect": []string{"/tab/2"}}
	req := httptest.NewRequest("POST", "/signup/sql", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	if err := SqlSignupAction(w, req); err != nil && err != ErrHandled {
		t.Fatalf("signup action: %v", err)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/login/sql?redirect=%2Ftab%2F2" {
		t.Fatalf("Expected redirect to login page with preserved redirect, got: %s", loc)
	}

	// Sign up again to trigger error
	req = httptest.NewRequest("POST", "/signup/sql", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	if err := SqlSignupAction(w, req); err != nil && err != ErrHandled {
		t.Fatalf("signup action error: %v", err)
	}

	loc = w.Result().Header.Get("Location")
	if loc != "/login/sql?error=exists&redirect=%2Ftab%2F2" {
		t.Fatalf("Expected redirect to login page with preserved error and redirect, got: %s", loc)
	}

	// Login successfully
	form = url.Values{"username": []string{"bob"}, "password": []string{"secret"}, "redirect": []string{"/tab/2?page=3"}}
	req = httptest.NewRequest("POST", "/login/sql", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	if err := SqlLoginAction(w, req); err != nil && err != ErrHandled {
		t.Fatalf("login action: %v", err)
	}

	// Read the new session from the response cookies
	req2_check := httptest.NewRequest("GET", "/", nil)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("No cookies returned from SqlLoginAction!")
	}
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
		t.Fatalf("Expected session redirect to be set to /tab/2?page=3, got: %v", session2.Values["Redirect"])
	}



	// Login failure
	form = url.Values{"username": []string{"bob"}, "password": []string{"wrong"}, "redirect": []string{"/tab/2?page=3"}}
	req = httptest.NewRequest("POST", "/login/sql", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	if err := SqlLoginAction(w, req); err != nil && err != ErrHandled {
		t.Fatalf("login action: %v", err)
	}

	loc = w.Result().Header.Get("Location")
	if loc != "/login/sql?error=invalid&redirect=%2Ftab%2F2%3Fpage%3D3" {
		t.Fatalf("Expected redirect to login page with preserved error and redirect, got: %s", loc)
	}
}
