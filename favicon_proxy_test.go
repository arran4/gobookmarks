package gobookmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper server providing favicon
func newFaviconServer(_ *testing.T, icon []byte) (*httptest.Server, *int) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(
			[]byte("<link rel='icon' href='/favicon.ico'>"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(
			icon)
	})
	return httptest.NewServer(mux), &hits
}

func TestFaviconDiskCacheExpiry(t *testing.T) {
	icon := []byte{0x89, 'P', 'N', 'G'}
	srv, hits := newFaviconServer(t, icon)
	defer srv.Close()

	cfg := &Configuration{
		FaviconCacheDir:  t.TempDir(),
		FaviconCacheSize: 1024 * 1024,
	}
	proxy := NewFaviconProxy(cfg)
	currTime := time.Now()
	proxy.now = func() time.Time { return currTime }

	req := httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)
	if *hits != 1 {
		t.Fatalf("expected 1 hit, got %d", *hits)
	}

	req = httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w = httptest.NewRecorder()
	proxy.ServeHTTP(w, req)
	if *hits != 1 {
		t.Fatalf("expected cache hit, hits %d", *hits)
	}

	currTime = currTime.Add(1500 * time.Millisecond)
	req = httptest.NewRequest("GET", "/proxy/favicon?url="+srv.URL, nil)
	w = httptest.NewRecorder()
	proxy.ServeHTTP(w, req)
	if *hits != 2 {
		t.Fatalf("expected refetch after expiry, hits %d", *hits)
	}
}

func TestFaviconMaxCacheCount(t *testing.T) {
	icon := []byte{0x89, 'P', 'N', 'G'}
	cfg := &Configuration{
		FaviconMaxCacheCount: 2,
	}
	proxy := NewFaviconProxy(cfg)

	// Add 3 items
	proxy.cacheFavicon("url1", icon, "image/png")
	proxy.cacheFavicon("url2", icon, "image/png")
	proxy.cacheFavicon("url3", icon, "image/png")

	proxy.mu.RLock()
	defer proxy.mu.RUnlock()

	if len(proxy.cache) > 2 {
		t.Fatalf("expected cache size <= 2, got %d", len(proxy.cache))
	}
}
