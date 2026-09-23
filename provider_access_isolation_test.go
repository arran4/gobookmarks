package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
	"testing"
)

func TestRequestCacheIsolation(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	mockP1 := &mockAccessProvider{
		nameFunc: func() string { return "prov1" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data from prov1", "sha1", nil
		},
	}
	RegisterProvider(mockP1)

	mockP2 := &mockAccessProvider{
		nameFunc: func() string { return "prov2" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data from prov2", "sha2", nil
		},
	}
	RegisterProvider(mockP2)

	cd := &CoreData{
		UserRef:      "testuser",
		requestCache: &requestCache{data: make(map[string]*bookmarkCacheEntry)},
	}

	ctx1 := context.WithValue(context.Background(), ContextValues("coreData"), cd)
	ctx1 = context.WithValue(ctx1, ContextValues("provider"), "prov1")

	ctx2 := context.WithValue(context.Background(), ContextValues("coreData"), cd)
	ctx2 = context.WithValue(ctx2, ContextValues("provider"), "prov2")

	b1, sha1, err1 := GetBookmarks(ctx1, "testuser", "main", nil)
	if err1 != nil {
		t.Fatalf("Unexpected error for prov1: %v", err1)
	}

	b2, sha2, err2 := GetBookmarks(ctx2, "testuser", "main", nil)
	if err2 != nil {
		t.Fatalf("Unexpected error for prov2: %v", err2)
	}

	if b1 == b2 {
		t.Fatalf("Cache isolation failed: b1=%q b2=%q", b1, b2)
	}
	if b1 != "data from prov1" || sha1 != "sha1" {
		t.Fatalf("Unexpected prov1 values: b1=%q sha1=%q", b1, sha1)
	}
	if b2 != "data from prov2" || sha2 != "sha2" {
		t.Fatalf("Unexpected prov2 values: b2=%q sha2=%q", b2, sha2)
	}

	if mockP1.getBookmarksCalls != 1 {
		t.Fatalf("Expected 1 call to prov1, got %d", mockP1.getBookmarksCalls)
	}
	if mockP2.getBookmarksCalls != 1 {
		t.Fatalf("Expected 1 call to prov2, got %d", mockP2.getBookmarksCalls)
	}

	// Ensure duplicate requests leverage cache properly
	_, _, err3 := GetBookmarks(ctx1, "testuser", "main", nil)
	if err3 != nil {
		t.Fatalf("Unexpected error for prov1 cached hit: %v", err3)
	}
	if mockP1.getBookmarksCalls != 1 {
		t.Fatalf("Expected still 1 call to prov1 due to caching, got %d", mockP1.getBookmarksCalls)
	}
}
