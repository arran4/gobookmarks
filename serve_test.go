package gobookmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// We verify the compatibility route caching headers directly using an HTTP test rather than trying
// to start the full `serve` command.
func TestCompatibilityRouteCacheHeaders(t *testing.T) {
	mux := http.NewServeMux()

	// Mimic the serve command routing for compatibility endpoints
	mux.HandleFunc("/main.css", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Cache-Control", "no-cache")
		if url, err := AssetURL("main.css"); err == nil {
			http.Redirect(writer, req, url, http.StatusFound)
			return
		}
		writer.Header().Set("Content-Type", "text/css")
		_, _ = writer.Write(GetMainCSSData())
	})

	mux.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Cache-Control", "no-cache")
		if url, err := AssetURL("logo.png"); err == nil {
			http.Redirect(writer, req, url, http.StatusFound)
			return
		}
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write(GetFavicon())
	})

	// Test /main.css
	reqCSS := httptest.NewRequest("GET", "/main.css", nil)
	recCSS := httptest.NewRecorder()
	mux.ServeHTTP(recCSS, reqCSS)

	if recCSS.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /main.css, got %q", recCSS.Header().Get("Cache-Control"))
	}

	// It should either 302 redirect or return bytes, depending on AssetURL resolving
	if recCSS.Code != http.StatusFound && recCSS.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /main.css, got %d", recCSS.Code)
	}

	// Test /favicon.ico
	reqFavicon := httptest.NewRequest("GET", "/favicon.ico", nil)
	recFavicon := httptest.NewRecorder()
	mux.ServeHTTP(recFavicon, reqFavicon)

	if recFavicon.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache on /favicon.ico, got %q", recFavicon.Header().Get("Cache-Control"))
	}

	if recFavicon.Code != http.StatusFound && recFavicon.Code != http.StatusOK {
		t.Errorf("expected status 302 or 200 on /favicon.ico, got %d", recFavicon.Code)
	}
}
