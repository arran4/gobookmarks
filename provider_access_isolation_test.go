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

	b1, _, _ := GetBookmarks(ctx1, "testuser", "main", nil)
	b2, _, _ := GetBookmarks(ctx2, "testuser", "main", nil)

	if b1 == b2 {
		t.Fatalf("Cache isolation failed: b1=%q b2=%q", b1, b2)
	}
}
