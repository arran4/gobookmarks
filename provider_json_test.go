package gobookmarks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
)

func TestJSONProviderCreateAndGet(t *testing.T) {
	tmp := t.TempDir()
	JSONDBPath = tmp
	p := JSONProvider{}
	user := "alice"
	text := "Category: Test\nhttp://example.com test"
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", text); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	got, sha, err := p.GetBookmarks(context.Background(), user, "main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	if got != text {
		t.Fatalf("expected %q got %q", text, got)
	}
	wantSha := sha256.Sum256([]byte(text))
	if sha != hex.EncodeToString(wantSha[:]) {
		t.Fatalf("unexpected sha")
	}
}

func TestJSONUserDirHash(t *testing.T) {
	JSONDBPath = "/base"
	path := userDir("../../etc/passwd")
	if filepath.Dir(path) != JSONDBPath {
		t.Fatalf("path escaped base: %s", path)
	}
	if filepath.Base(path) == "../../etc/passwd" {
		t.Fatalf("user dir not hashed")
	}
}
