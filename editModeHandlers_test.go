package gobookmarks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestEditModeToggle(t *testing.T) {
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
	ctx = context.WithValue(ctx, ContextValues("configuration"), config)
	req = req.WithContext(ctx)

	// enable edit mode
	w = httptest.NewRecorder()
	if err := StartEditMode(w, req); err != nil {
		t.Fatalf("StartEditMode: %v", err)
	}
	if req.URL.Query().Get("edit") != "1" {
		t.Fatalf("edit mode query not set")
	}

	var cd *CoreData
	handler := config.CoreAdderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cd = r.Context().Value(ContextValues("coreData")).(*CoreData)
	}))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if cd == nil || !cd.EditMode {
		t.Fatalf("EditMode flag not propagated via middleware")
	}

	// disable edit mode
	w = httptest.NewRecorder()
	if err := StopEditMode(w, req); err != nil {
		t.Fatalf("StopEditMode: %v", err)
	}
	if req.URL.Query().Get("edit") != "" {
		t.Fatalf("edit mode flag should be cleared")
	}

	cd = nil
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if cd == nil || cd.EditMode {
		t.Fatalf("EditMode flag should be false after disabling")
	}
}
