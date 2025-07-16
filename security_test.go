package gobookmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	ContentSecurityPolicy = "img-src example.com"
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Result().Header.Get("Content-Security-Policy"); got != ContentSecurityPolicy {
		t.Fatalf("expected %q got %q", ContentSecurityPolicy, got)
	}
}
