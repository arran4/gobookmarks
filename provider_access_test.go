package gobookmarks

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/oauth2"
)

type MockAccessProvider struct {
	NameFunc            func() string
	GetBookmarksFunc    func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error)
	UpdateBookmarksFunc func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	CreateBookmarksFunc func(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	GetTagsFunc         func(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error)
	GetBranchesFunc     func(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	GetCommitsFunc      func(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error)

	GetBookmarksCalls    int
	UpdateBookmarksCalls int
	CreateBookmarksCalls int
}

func (m *MockAccessProvider) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock_access"
}

func (m *MockAccessProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return nil
}

func (m *MockAccessProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return &User{Login: "testuser"}, nil
}

func (m *MockAccessProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	return nil
}

func (m *MockAccessProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	return true, nil
}

func (m *MockAccessProvider) DefaultServer() string {
	return ""
}

func (m *MockAccessProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	m.GetBookmarksCalls++
	if m.GetBookmarksFunc != nil {
		return m.GetBookmarksFunc(ctx, user, ref, token)
	}
	return "", "", nil
}

func (m *MockAccessProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	m.UpdateBookmarksCalls++
	if m.UpdateBookmarksFunc != nil {
		return m.UpdateBookmarksFunc(ctx, user, token, sourceRef, branch, text, expectSHA)
	}
	return nil
}

func (m *MockAccessProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	m.CreateBookmarksCalls++
	if m.CreateBookmarksFunc != nil {
		return m.CreateBookmarksFunc(ctx, user, token, branch, text)
	}
	return nil
}

func (m *MockAccessProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	if m.GetTagsFunc != nil {
		return m.GetTagsFunc(ctx, user, token)
	}
	return nil, nil
}

func (m *MockAccessProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	if m.GetBranchesFunc != nil {
		return m.GetBranchesFunc(ctx, user, token)
	}
	return nil, nil
}

func (m *MockAccessProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	if m.GetCommitsFunc != nil {
		return m.GetCommitsFunc(ctx, user, token, ref, page, perPage)
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
	mockP := &MockAccessProvider{
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "bookmarks data", "sha123", nil
		},
	}
	RegisterProvider(mockP)
	defer func() {
		// Clean up the registered mock provider so it doesn't affect other tests.
		// RegisterProvider adds to a global slice. Let's just restore it.
		// A safer way is to just set ProviderOrder or reset it, but since we append,
		// we can just let it be or use a unique name.
	}()
	// Using a unique name
	mockP.NameFunc = func() string { return "mock_reuse" }
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_reuse")

	// First call should hit the provider
	b1, sha1, err1 := GetBookmarks(ctx, "testuser", "main", nil)
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	if b1 != "bookmarks data" || sha1 != "sha123" {
		t.Errorf("unexpected results: %q, %q", b1, sha1)
	}
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call, got %d", mockP.GetBookmarksCalls)
	}

	// Second call with the SAME context (and thus same requestCache) should NOT hit the provider
	b2, sha2, err2 := GetBookmarks(ctx, "testuser", "main", nil)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if b2 != "bookmarks data" || sha2 != "sha123" {
		t.Errorf("unexpected results: %q, %q", b2, sha2)
	}
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected still 1 call, got %d", mockP.GetBookmarksCalls)
	}
}

func TestIndependentRequestCaches(t *testing.T) {
	mockP := &MockAccessProvider{
		NameFunc: func() string { return "mock_indep" },
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "bookmarks data indep", "sha456", nil
		},
	}
	RegisterProvider(mockP)

	ctx1 := setupTestContext("testuser", "mock_indep")
	ctx2 := setupTestContext("testuser", "mock_indep")

	// Call with first context
	_, _, _ = GetBookmarks(ctx1, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call, got %d", mockP.GetBookmarksCalls)
	}

	// Call with second context should hit the provider again
	_, _, _ = GetBookmarks(ctx2, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 2 {
		t.Errorf("expected 2 calls, got %d", mockP.GetBookmarksCalls)
	}
}

func TestMutationInvalidatesCache(t *testing.T) {
	bookmarksState := "initial bookmarks"
	shaState := "initialSha"

	mockP := &MockAccessProvider{
		NameFunc: func() string { return "mock_mutate" },
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return bookmarksState, shaState, nil
		},
		UpdateBookmarksFunc: func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
			bookmarksState = text
			shaState = "newSha"
			return nil
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_mutate")

	// Initial fetch
	b1, _, _ := GetBookmarks(ctx, "testuser", "main", nil)
	if b1 != "initial bookmarks" {
		t.Errorf("expected 'initial bookmarks', got %q", b1)
	}

	// Should be cached now
	_, _, _ = GetBookmarks(ctx, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call, got %d", mockP.GetBookmarksCalls)
	}

	// Mutate
	err := UpdateBookmarks(ctx, "testuser", nil, "main", "main", "updated bookmarks", "initialSha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockP.UpdateBookmarksCalls != 1 {
		t.Errorf("expected 1 update call, got %d", mockP.UpdateBookmarksCalls)
	}

	// Read again, should hit provider
	b3, sha3, _ := GetBookmarks(ctx, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 2 {
		t.Errorf("expected 2 calls, got %d", mockP.GetBookmarksCalls)
	}
	if b3 != "updated bookmarks" || sha3 != "newSha" {
		t.Errorf("expected updated state, got %q, %q", b3, sha3)
	}
}

func TestFailedMutationPreservesCache(t *testing.T) {
	mockP := &MockAccessProvider{
		NameFunc: func() string { return "mock_fail_mutate" },
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "preserved bookmarks", "presSha", nil
		},
		UpdateBookmarksFunc: func(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
			return errors.New("simulated conflict")
		},
	}
	RegisterProvider(mockP)

	ctx := setupTestContext("testuser", "mock_fail_mutate")

	// Initial fetch
	_, _, _ = GetBookmarks(ctx, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call, got %d", mockP.GetBookmarksCalls)
	}

	// Failed mutation
	err := UpdateBookmarks(ctx, "testuser", nil, "main", "main", "failed update", "presSha")
	if err == nil {
		t.Fatal("expected error on update, got nil")
	}

	// Read again, should NOT hit provider because cache is preserved on failure
	b, sha, _ := GetBookmarks(ctx, "testuser", "main", nil)
	if mockP.GetBookmarksCalls != 1 {
		t.Errorf("expected still 1 call, got %d", mockP.GetBookmarksCalls)
	}
	if b != "preserved bookmarks" || sha != "presSha" {
		t.Errorf("expected preserved state, got %q, %q", b, sha)
	}
}

func TestProviderSelectionFromContext(t *testing.T) {
	mockP1 := &MockAccessProvider{
		NameFunc: func() string { return "mock_select_1" },
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data1", "sha1", nil
		},
	}
	RegisterProvider(mockP1)

	mockP2 := &MockAccessProvider{
		NameFunc: func() string { return "mock_select_2" },
		GetBookmarksFunc: func(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
			return "data2", "sha2", nil
		},
	}
	RegisterProvider(mockP2)

	ctx1 := setupTestContext("testuser", "mock_select_1")
	ctx2 := setupTestContext("testuser", "mock_select_2")

	b1, _, _ := GetBookmarks(ctx1, "testuser", "main", nil)
	if b1 != "data1" {
		t.Errorf("expected data1, got %q", b1)
	}

	b2, _, _ := GetBookmarks(ctx2, "testuser", "main", nil)
	if b2 != "data2" {
		t.Errorf("expected data2, got %q", b2)
	}

	if mockP1.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call for p1, got %d", mockP1.GetBookmarksCalls)
	}
	if mockP2.GetBookmarksCalls != 1 {
		t.Errorf("expected 1 call for p2, got %d", mockP2.GetBookmarksCalls)
	}
}
