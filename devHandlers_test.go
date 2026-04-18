package gobookmarks

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestCSSColumnToggle(t *testing.T) {
	Config.SessionName = "testsession"
	SessionStore = sessions.NewCookieStore([]byte("secret"))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	session, err := getSession(w, req)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	req = req.WithContext(ctx)

	w = httptest.NewRecorder()
	if err := EnableCSSColumnsAction(w, req); err != nil {
		t.Fatalf("EnableCSSColumnsAction: %v", err)
	}
	if v, ok := session.Values["useCSSColumns"].(bool); !ok || !v {
		t.Fatalf("cssColumns not enabled in session")
	}

	w = httptest.NewRecorder()
	if err := DisableCSSColumnsAction(w, req); err != nil {
		t.Fatalf("DisableCSSColumnsAction: %v", err)
	}
	if v, ok := session.Values["useCSSColumns"].(bool); !ok || v {
		t.Fatalf("cssColumns not disabled in session")
	}
}
