package gobookmarks

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestCssColumnToggle(t *testing.T) {
	config := NewConfiguration()
	config.SessionName = "testsession"
	config.SessionStore = sessions.NewCookieStore([]byte("secret"))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	session, err := config.getSession(w, req)
	if err != nil {
		t.Fatalf("getSession: %v", err)
	}
	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	req = req.WithContext(ctx)

	w = httptest.NewRecorder()
	if err := EnableCssColumnsAction(w, req); err != nil {
		t.Fatalf("EnableCssColumnsAction: %v", err)
	}
	if v, ok := session.Values["useCssColumns"].(bool); !ok || !v {
		t.Fatalf("cssColumns not enabled in session")
	}

	w = httptest.NewRecorder()
	if err := DisableCssColumnsAction(w, req); err != nil {
		t.Fatalf("DisableCssColumnsAction: %v", err)
	}
	if v, ok := session.Values["useCssColumns"].(bool); !ok || v {
		t.Fatalf("cssColumns not disabled in session")
	}
}
