package gobookmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSPMiddleware(t *testing.T) {
	h := CSPMiddleware("default-src 'self'")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Fatalf("unexpected header: %q", got)
	}
}
