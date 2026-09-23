package gobookmarks

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
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
		// Use real template logic simulating a real page with multiple reads
		tmplStr := `{{ tabName }} {{ bookmarks }} {{ tabName }}`
		tmpl, err := template.New("test").Funcs(NewFuncs(r)).Parse(tmplStr)
		if err != nil {
			t.Fatalf("Template parse failed: %v", err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, r.Context().Value(ContextValues("coreData")).(*CoreData))
		if err != nil {
			t.Fatalf("Template execute failed: %v", err)
		}
		w.Write(buf.Bytes())
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
		tmplStr := `{{ tabName }} {{ bookmarks }} {{ tabName }}`
		tmpl, err := template.New("test").Funcs(NewFuncs(r)).Parse(tmplStr)
		if err != nil {
			t.Fatalf("Template parse failed: %v", err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, r.Context().Value(ContextValues("coreData")).(*CoreData))
		if err != nil {
			t.Fatalf("Template execute failed: %v", err)
		}
		w.Write(buf.Bytes())
	}).ServeHTTP(wNoCache, reqNoCache)

	uncachedCount := callCount
	t.Logf("Uncached path made %d provider calls", uncachedCount)

	if uncachedCount != 3 {
		t.Errorf("Expected exactly 3 uncached calls, got %d", uncachedCount)
	}

	if w.Body.String() != wNoCache.Body.String() {
		t.Errorf("Output mismatch between cached and uncached rendering.\nCached: %q\nUncached: %q", w.Body.String(), wNoCache.Body.String())
	}
}
