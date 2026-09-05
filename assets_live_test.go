//go:build live

package gobookmarks

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveAssetReloading(t *testing.T) {
	dir := t.TempDir()

	// Write initial CSS
	cssContent1 := []byte("body { background: #fff; }")
	err := os.WriteFile(filepath.Join(dir, "main.css"), cssContent1, 0644)
	if err != nil {
		t.Fatalf("failed to write initial css: %v", err)
	}
	// Also need logo.png for it to be found
	err = os.WriteFile(filepath.Join(dir, "logo.png"), []byte("png"), 0644)
	if err != nil {
		t.Fatalf("failed to write initial png: %v", err)
	}

	reg, err := NewRegistry(os.DirFS(dir), true)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	liveReg := &LiveRegistryWrapper{reg: reg}

	// 1. Get initial asset URL
	url1, err := liveReg.AssetURL("main.css")
	if err != nil {
		t.Fatalf("AssetURL failed: %v", err)
	}
	hash1 := sha256.Sum256(cssContent1)
	hex1 := hex.EncodeToString(hash1[:])
	if !strings.Contains(url1, hex1) {
		t.Errorf("url1 %q does not contain hash1 %q", url1, hex1)
	}

	// Verify headers for live asset
	req := httptest.NewRequest("GET", url1, nil)
	rec := httptest.NewRecorder()
	liveReg.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control: no-store, got %q", rec.Header().Get("Cache-Control"))
	}

	// 2. Modify the CSS file on disk
	cssContent2 := []byte("body { background: #000; }")
	err = os.WriteFile(filepath.Join(dir, "main.css"), cssContent2, 0644)
	if err != nil {
		t.Fatalf("failed to write updated css: %v", err)
	}

	// 3. Get asset URL again and expect it to change because of LiveRegistryWrapper
	url2, err := liveReg.AssetURL("main.css")
	if err != nil {
		t.Fatalf("AssetURL failed after modification: %v", err)
	}
	hash2 := sha256.Sum256(cssContent2)
	hex2 := hex.EncodeToString(hash2[:])
	if !strings.Contains(url2, hex2) {
		t.Errorf("url2 %q does not contain hash2 %q", url2, hex2)
	}

	if url1 == url2 {
		t.Errorf("URL did not change after modifying asset in live mode")
	}

	// 4. Request the new asset URL and verify content
	req2 := httptest.NewRequest("GET", url2, nil)
	rec2 := httptest.NewRecorder()
	liveReg.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 OK for new URL, got %d", rec2.Code)
	}
	if rec2.Body.String() != string(cssContent2) {
		t.Errorf("expected updated content %q, got %q", string(cssContent2), rec2.Body.String())
	}
}

func TestGetAssetRegistryLive(t *testing.T) {
	// Should not panic in live mode
	reg := GetAssetRegistry()
	if reg == nil {
		t.Fatal("GetAssetRegistry returned nil")
	}

	// Should also work
	prov := GetAssetProvider()
	if prov == nil {
		t.Fatal("GetAssetProvider returned nil")
	}
}

func TestLiveReloadFailurePropagation(t *testing.T) {
	dir := t.TempDir()

	// Initially all files exist so initialization succeeds
	err := os.WriteFile(filepath.Join(dir, "main.css"), []byte("css"), 0644)
	if err != nil {
		t.Fatalf("failed to write initial css: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "logo.png"), []byte("png"), 0644)
	if err != nil {
		t.Fatalf("failed to write initial png: %v", err)
	}

	reg, err := NewRegistry(os.DirFS(dir), true)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	liveReg := &LiveRegistryWrapper{reg: reg}

	// Now let's remove permissions to cause a read failure
	err = os.Chmod(filepath.Join(dir, "main.css"), 0000)
	if err != nil {
		t.Fatalf("failed to change permissions: %v", err)
	}
	// We need to ensure we clean up so temp dir can be deleted
	defer os.Chmod(filepath.Join(dir, "main.css"), 0644)

	// AssetURL should propagate the reload error
	_, err = liveReg.AssetURL("main.css")
	if err == nil {
		t.Fatalf("AssetURL expected to fail due to unreadable file")
	}

	// ServeHTTP should return 500
	req := httptest.NewRequest("GET", "/assets/main.css", nil)
	rec := httptest.NewRecorder()
	liveReg.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("ServeHTTP expected to return 500 on reload failure, got %d", rec.Code)
	}
}
