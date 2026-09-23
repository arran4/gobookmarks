package gobookmarks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"golang.org/x/oauth2"
)

type mockAccessProvider struct {
	nameFunc            func() string
	getBookmarksFunc    func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error)
	updateBookmarksFunc func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	createBookmarksFunc func(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	getTagsFunc         func(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error)
	getBranchesFunc     func(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	getCommitsFunc      func(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error)

	getBookmarksCalls    int32
	updateBookmarksCalls int32
	createBookmarksCalls int32
}

func (m *mockAccessProvider) Name() string {
	if m.nameFunc != nil {
		return m.nameFunc()
	}
	return "mock_access"
}

func (m *mockAccessProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return nil
}

func (m *mockAccessProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return &User{Login: "testuser"}, nil
}

func (m *mockAccessProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	return nil
}

func (m *mockAccessProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	return true, nil
}

func (m *mockAccessProvider) DefaultServer() string {
	return ""
}

func (m *mockAccessProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	atomic.AddInt32(&m.getBookmarksCalls, 1)
	if m.getBookmarksFunc != nil {
		return m.getBookmarksFunc(ctx, user, ref, token)
	}
	return "", "", nil
}

func (m *mockAccessProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	atomic.AddInt32(&m.updateBookmarksCalls, 1)
	if m.updateBookmarksFunc != nil {
		return m.updateBookmarksFunc(ctx, user, token, sourceRef, branch, text, expectSHA)
	}
	return nil
}

func (m *mockAccessProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	atomic.AddInt32(&m.createBookmarksCalls, 1)
	if m.createBookmarksFunc != nil {
		return m.createBookmarksFunc(ctx, user, token, branch, text)
	}
	return nil
}

func (m *mockAccessProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	if m.getTagsFunc != nil {
		return m.getTagsFunc(ctx, user, token)
	}
	return nil, nil
}

func (m *mockAccessProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	if m.getBranchesFunc != nil {
		return m.getBranchesFunc(ctx, user, token)
	}
	return nil, nil
}

func (m *mockAccessProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	if m.getCommitsFunc != nil {
		return m.getCommitsFunc(ctx, user, token, ref, page, perPage)
	}
	return nil, nil
}

func setupTestContext(user string, providerName string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValues("provider"), providerName)
	ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{
		UserRef:      user,
		requestCache: &requestCache{data: make(map[string]*bookmarkCacheEntry)},
	})
	return ctx
}

func TestRequestCacheReuse(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_reuse" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "bookmarks data", "sha123", nil
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_reuse")

	b1, sha1, err1 := GetBookmarks(ctx, "testuser", "main", nil)
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	if b1 != "bookmarks data" || sha1 != "sha123" {
		t.Errorf("unexpected results: %q, %q", b1, sha1)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected 1 call, got %d", mockP.getBookmarksCalls)
	}

	b2, sha2, err2 := GetBookmarks(ctx, "testuser", "main", nil)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if b2 != "bookmarks data" || sha2 != "sha123" {
		t.Errorf("unexpected results: %q, %q", b2, sha2)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected still 1 call, got %d", mockP.getBookmarksCalls)
	}
}

func TestIndependentRequestCaches(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_indep" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "bookmarks data indep", "sha456", nil
		},
	}
	RegisterProvider(mockP)

	ctx1 := setupTestContext("testuser", "mock_indep")
	ctx2 := setupTestContext("testuser", "mock_indep")

	_, _, err := GetBookmarks(ctx1, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected 1 call, got %d", mockP.getBookmarksCalls)
	}

	_, _, err = GetBookmarks(ctx2, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(2) {
		t.Errorf("expected 2 calls, got %d", mockP.getBookmarksCalls)
	}
}

func TestMutationInvalidatesCache(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	bookmarksState := "initial bookmarks"
	shaState := "initialSha"

	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_mutate" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return bookmarksState, shaState, nil
		},
		updateBookmarksFunc: func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
			bookmarksState = text
			shaState = "newSha"
			return nil
		},
		createBookmarksFunc: func(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
			bookmarksState = text
			shaState = "newShaCreated"
			return nil
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_mutate")

	b1, _, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b1 != "initial bookmarks" {
		t.Errorf("expected 'initial bookmarks', got %q", b1)
	}

	_, _, err = GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected 1 call, got %d", mockP.getBookmarksCalls)
	}

	err = UpdateBookmarks(ctx, "testuser", nil, "main", "main", "updated bookmarks", "initialSha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.updateBookmarksCalls != int32(1) {
		t.Errorf("expected 1 update call, got %d", mockP.updateBookmarksCalls)
	}

	b3, sha3, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(2) {
		t.Errorf("expected 2 calls, got %d", mockP.getBookmarksCalls)
	}
	if b3 != "updated bookmarks" || sha3 != "newSha" {
		t.Errorf("expected updated state, got %q, %q", b3, sha3)
	}

	// Test CreateBookmarks invalidation
	err = CreateBookmarks(ctx, "testuser", nil, "main", "created bookmarks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.createBookmarksCalls != int32(1) {
		t.Errorf("expected 1 create call, got %d", mockP.createBookmarksCalls)
	}

	b4, sha4, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(3) {
		t.Errorf("expected 3 calls, got %d", mockP.getBookmarksCalls)
	}
	if b4 != "created bookmarks" || sha4 != "newShaCreated" {
		t.Errorf("expected created state, got %q, %q", b4, sha4)
	}
}

func TestFailedMutationPreservesCache(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_fail_mutate" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "preserved bookmarks", "presSha", nil
		},
		updateBookmarksFunc: func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
			return errors.New("simulated conflict")
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_fail_mutate")

	_, _, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected 1 call, got %d", mockP.getBookmarksCalls)
	}

	err = UpdateBookmarks(ctx, "testuser", nil, "main", "main", "failed update", "presSha")
	if err == nil {
		t.Fatal("expected error on update, got nil")
	}

	b, sha, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected still 1 call, got %d", mockP.getBookmarksCalls)
	}
	if b != "preserved bookmarks" || sha != "presSha" {
		t.Errorf("expected preserved state, got %q, %q", b, sha)
	}
}

func TestProviderSelectionFromContext(t *testing.T) {
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
		nameFunc: func() string { return "mock_select_1" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data1", "sha1", nil
		},
	}
	RegisterProvider(mockP1)

	mockP2 := &mockAccessProvider{
		nameFunc: func() string { return "mock_select_2" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data2", "sha2", nil
		},
	}
	RegisterProvider(mockP2)

	ctx1 := setupTestContext("testuser", "mock_select_1")
	ctx2 := setupTestContext("testuser", "mock_select_2")

	b1, _, err := GetBookmarks(ctx1, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b1 != "data1" {
		t.Errorf("expected data1, got %q", b1)
	}

	b2, _, err := GetBookmarks(ctx2, "testuser", "main", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b2 != "data2" {
		t.Errorf("expected data2, got %q", b2)
	}

	if mockP1.getBookmarksCalls != 1 {
		t.Errorf("expected 1 call for p1, got %d", mockP1.getBookmarksCalls)
	}
	if mockP2.getBookmarksCalls != 1 {
		t.Errorf("expected 1 call for p2, got %d", mockP2.getBookmarksCalls)
	}
}

func TestFailedReadNotCached(t *testing.T) {
	origOrder := ProviderNames()
	origProviders := make(map[string]Provider)
	for _, name := range origOrder {
		origProviders[name] = GetProvider(name)
	}
	defer func() {
		providers = origProviders
		SetProviderOrder(origOrder)
	}()

	mockP := &mockAccessProvider{
		nameFunc: func() string { return "mock_fail_read" },
		getBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "", "", errors.New("simulated read error")
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_fail_read")

	_, _, err := GetBookmarks(ctx, "testuser", "main", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mockP.getBookmarksCalls != int32(1) {
		t.Errorf("expected 1 call, got %d", mockP.getBookmarksCalls)
	}

	// Call again, should NOT be cached
	_, _, err = GetBookmarks(ctx, "testuser", "main", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mockP.getBookmarksCalls != int32(2) {
		t.Errorf("expected 2 calls, got %d", mockP.getBookmarksCalls)
	}
}
