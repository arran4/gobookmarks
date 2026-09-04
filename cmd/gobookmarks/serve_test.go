package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// We verify the compatibility route caching headers directly using an HTTP test rather than trying
// to start the full `serve` command.
func TestCompatibilityRouteCacheHeaders(t *testing.T) {
	// setupRouter builds the production router exactly as the serve command uses it
	r := setupRouter()

	// Test /main.css
	reqCSS := httptest.NewRequest("GET", "/main.css", nil)
	recCSS := httptest.NewRecorder()
	r.ServeHTTP(recCSS, reqCSS)

	if recCSS.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /main.css, got %q", recCSS.Header().Get("Cache-Control"))
	}

	if recCSS.Code != http.StatusFound && recCSS.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /main.css, got %d", recCSS.Code)
	}

	// Test /favicon.ico
	reqFavicon := httptest.NewRequest("GET", "/favicon.ico", nil)
	recFavicon := httptest.NewRecorder()
	r.ServeHTTP(recFavicon, reqFavicon)

	if recFavicon.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /favicon.ico, got %q", recFavicon.Header().Get("Cache-Control"))
	}

	if recFavicon.Code != http.StatusFound && recFavicon.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /favicon.ico, got %d", recFavicon.Code)
	}
}
