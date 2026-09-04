package gobookmarks

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestAssetRegistry(t *testing.T) {
	cssContent := []byte("body { background: #fff; }")
	pngContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	mockFS := fstest.MapFS{
		"main.css": &fstest.MapFile{Data: cssContent},
		"logo.png": &fstest.MapFile{Data: pngContent},
	}

	reg, err := NewRegistry(mockFS, false)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	cssHash := sha256.Sum256(cssContent)
	cssHex := hex.EncodeToString(cssHash[:])
	expectedCSSURL := "/assets/main." + cssHex + ".css"

	pngHash := sha256.Sum256(pngContent)
	pngHex := hex.EncodeToString(pngHash[:])
	expectedPNGURL := "/assets/logo." + pngHex + ".png"

	url, err := reg.AssetURL("main.css")
	if err != nil {
		t.Fatalf("AssetURL(main.css) failed: %v", err)
	}
	if url != expectedCSSURL {
		t.Errorf("expected %s, got %s", expectedCSSURL, url)
	}

	url, err = reg.AssetURL("/main.css")
	if err != nil {
		t.Fatalf("AssetURL(/main.css) failed: %v", err)
	}
	if url != expectedCSSURL {
		t.Errorf("expected %s, got %s", expectedCSSURL, url)
	}

	url, err = reg.AssetURL("logo.png")
	if err != nil {
		t.Fatalf("AssetURL(logo.png) failed: %v", err)
	}
	if url != expectedPNGURL {
		t.Errorf("expected %s, got %s", expectedPNGURL, url)
	}

	// Unknown asset should fail
	_, err = reg.AssetURL("unknown.js")
	if err == nil {
		t.Errorf("expected error for unknown asset, got nil")
	}

	// Test ServeHTTP - 200 OK with caching headers
	req := httptest.NewRequest("GET", expectedCSSURL, nil)
	rec := httptest.NewRecorder()
	reg.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("expected Cache-Control header 'public, max-age=31536000, immutable', got '%s'", cc)
	}
	expectedETag := `"` + cssHex + `"`
	if etag := rec.Header().Get("ETag"); etag != expectedETag {
		t.Errorf("expected ETag '%s', got '%s'", expectedETag, etag)
	}
	if body := rec.Body.String(); body != string(cssContent) {
		t.Errorf("expected body '%s', got '%s'", string(cssContent), body)
	}

	// Test Conditional Request - 304 Not Modified
	req304 := httptest.NewRequest("GET", expectedCSSURL, nil)
	req304.Header.Set("If-None-Match", expectedETag)
	rec304 := httptest.NewRecorder()
	reg.ServeHTTP(rec304, req304)

	if rec304.Code != http.StatusNotModified {
		t.Errorf("expected status 304 for conditional request, got %d", rec304.Code)
	}

	// Test 404 Not Found
	req404 := httptest.NewRequest("GET", "/assets/non-existent.css", nil)
	rec404 := httptest.NewRecorder()
	reg.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec404.Code)
	}
}

func TestNonPublicAssets(t *testing.T) {
	mockFS := fstest.MapFS{
		"main.css":   &fstest.MapFile{Data: []byte("css")},
		"logo.png":   &fstest.MapFile{Data: []byte("png")},
		"secret.txt": &fstest.MapFile{Data: []byte("secret")},
		"README.md":  &fstest.MapFile{Data: []byte("readme")},
	}

	reg, err := NewRegistry(mockFS, false)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Should succeed for public assets
	_, err = reg.AssetURL("main.css")
	if err != nil {
		t.Errorf("AssetURL(main.css) failed: %v", err)
	}

	// Should fail for non-public assets even if they exist in FS
	_, err = reg.AssetURL("secret.txt")
	if err == nil {
		t.Errorf("expected error for non-public asset secret.txt, got nil")
	}

	_, err = reg.AssetURL("README.md")
	if err == nil {
		t.Errorf("expected error for non-public asset README.md, got nil")
	}

	// Also verify ServeHTTP handles non-public correctly by returning 404, not exposing them
	secretHash := sha256.Sum256([]byte("secret"))
	secretHex := hex.EncodeToString(secretHash[:])
	req := httptest.NewRequest("GET", "/assets/secret."+secretHex+".txt", nil)
	rec := httptest.NewRecorder()
	reg.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-public asset, got %d", rec.Code)
	}
}

func TestGlobalAssetRegistry(t *testing.T) {
	cssURL, err := AssetURL("main.css")
	if err != nil {
		t.Fatalf("AssetURL(main.css) failed: %v", err)
	}
	if cssURL == "" || cssURL == "/main.css" {
		t.Errorf("expected fingerprinted asset URL, got %s", cssURL)
	}

	logoURL, err := AssetURL("logo.png")
	if err != nil {
		t.Fatalf("AssetURL(logo.png) failed: %v", err)
	}
	if logoURL == "" || logoURL == "/logo.png" {
		t.Errorf("expected fingerprinted asset URL, got %s", logoURL)
	}
}
