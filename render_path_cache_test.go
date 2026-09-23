package gobookmarks

import (
	"context"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRenderPathCacheCount(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	callCount := 0
	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_render" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			callCount++
			return "Category: Test\nhttp://example.com test", "sha123", nil
		},
	}
	RegisterProvider(mockP)

	req := httptest.NewRequest("GET", "/?ref=main", nil)

	// Create minimal session
	store := sessions.NewCookieStore([]byte("secret"))
	session, _ := store.Get(req, Config.GetSessionName())
	session.Values["GithubUser"] = &User{Login: "testuser"}
	session.Values["Provider"] = "mock_render"

	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	req = req.WithContext(ctx)

	handler := CoreAdderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate func usages inside template rendering
		GetBookmarks(r.Context(), "testuser", "main", nil) // simulated tabName call
		GetBookmarks(r.Context(), "testuser", "main", nil) // simulated bookmark page rendering
		GetBookmarks(r.Context(), "testuser", "main", nil) // simulated activeTab evaluation

		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	cachedCount := callCount
	t.Logf("Cached path made %d provider calls", cachedCount)

	if cachedCount != 1 {
		t.Errorf("Expected exactly 1 call with cache, got %d", cachedCount)
	}

	// Now run it without cache by overriding the middleware manually
	callCount = 0
	reqNoCache := httptest.NewRequest("GET", "/?ref=main", nil)
	ctxNoCache := context.WithValue(reqNoCache.Context(), ContextValues("session"), session)

	// Bypass CoreAdderMiddleware entirely to not create a cache
	ctxNoCache = context.WithValue(ctxNoCache, ContextValues("provider"), "mock_render")
	ctxNoCache = context.WithValue(ctxNoCache, ContextValues("coreData"), &CoreData{
		UserRef:      "testuser",
		Title:        "test",
		Tab:          0,
		requestCache: nil, // DISABLE CACHE
	})

	reqNoCache = reqNoCache.WithContext(ctxNoCache)
	wNoCache := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetBookmarks(r.Context(), "testuser", "main", nil)
		GetBookmarks(r.Context(), "testuser", "main", nil)
		GetBookmarks(r.Context(), "testuser", "main", nil)
	}).ServeHTTP(wNoCache, reqNoCache)

	uncachedCount := callCount
	t.Logf("Uncached path made %d provider calls", uncachedCount)

	if uncachedCount <= cachedCount {
		t.Errorf("Expected uncached calls (%d) to be > cached calls (%d)", uncachedCount, cachedCount)
	}
}
