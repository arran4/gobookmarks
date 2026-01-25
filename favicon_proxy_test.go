package gobookmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper server providing favicon
func newFaviconServer(t *testing.T, icon []byte) (*httptest.Server, *int) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<link rel='icon' href='/favicon.ico'>"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("Content-Type", "image/png")
		w.Write(icon)
	})
	return httptest.NewServer(mux), &hits
}

func TestFaviconDiskCacheExpiry(t *testing.T) {
	icon := []byte{0x89, 'P', 'N', 'G'}
	srv, hits := newFaviconServer(t, icon)
	defer srv.Close()

	config := NewConfiguration()
	config.FaviconCacheDir = t.TempDir()
	config.FaviconCacheSize = 1024 * 1024
	config.FaviconMaxCacheCount = 1000
	svc := NewFaviconService(config)

	req := httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)
	if *hits != 1 {
		t.Fatalf("expected 1 hit, got %d", *hits)
	}

	req = httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w = httptest.NewRecorder()
	svc.ServeHTTP(w, req)
	if *hits != 1 {
		t.Fatalf("expected cache hit, hits %d", *hits)
	}

	time.Sleep(1500 * time.Millisecond)
	req = httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w = httptest.NewRecorder()
	svc.ServeHTTP(w, req)
	if *hits != 2 {
		t.Fatalf("expected refetch after expiry, hits %d", *hits)
	}
}

func TestFaviconMaxCacheCount(t *testing.T) {
	icon := []byte{0x89, 'P', 'N', 'G'}
	config := NewConfiguration()
	config.FaviconMaxCacheCount = 2
	svc := NewFaviconService(config)

	// Add 3 items
	svc.cacheFavicon("url1", icon, "image/png")
	svc.cacheFavicon("url2", icon, "image/png")
	svc.cacheFavicon("url3", icon, "image/png")

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if len(svc.cache) > 2 {
		t.Fatalf("expected cache size <= 2, got %d", len(svc.cache))
	}
}
