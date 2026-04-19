package gobookmarks

import (
	"context"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type MockProviderForAddTab struct {
	FileContents string
	Sha          string
}

func (m *MockProviderForAddTab) Name() string { return "Mock" }
func (m *MockProviderForAddTab) Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return nil
}
func (m *MockProviderForAddTab) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return nil, nil
}
func (m *MockProviderForAddTab) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	return nil, nil
}
func (m *MockProviderForAddTab) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	return nil, nil
}
func (m *MockProviderForAddTab) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	return nil, nil
}
func (m *MockProviderForAddTab) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	return m.FileContents, m.Sha, nil
}
func (m *MockProviderForAddTab) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	m.FileContents = text
	m.Sha = "new-sha"
	return nil
}
func (m *MockProviderForAddTab) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	return nil
}
func (m *MockProviderForAddTab) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	return nil
}
func (m *MockProviderForAddTab) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	return true, nil
}
func (m *MockProviderForAddTab) DefaultServer() string { return "" }

func TestTabAddIntegration(t *testing.T) {
	bookmarksStr := `Tab: Tab1
Page: P1

Tab: Tab2
Page: P2`

	provider := &MockProviderForAddTab{
		FileContents: bookmarksStr,
		Sha:          "mock-sha",
	}
	RegisterProvider(provider)

	session := &sessions.Session{
		Values: map[interface{}]interface{}{
			"GithubUser": &User{Login: "testuser"},
		},
	}

	form := url.Values{}
	form.Add("name", "NewTab")
	form.Add("text", "Page: P3")
	form.Add("task", TaskSaveAndStopEditing)

	req, _ := http.NewRequest("POST", "/editTab", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), ContextValues("session"), session)
	ctx = context.WithValue(ctx, ContextValues("provider"), "Mock")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	err := TabEditSaveAction(rr, req)
	if err != nil {
		t.Fatalf("TabEditSaveAction failed: %v", err)
	}

	// Verify the tab was added to the end and not replacing Tab1
	parsed := ParseBookmarks(provider.FileContents)
	if len(parsed) != 3 {
		t.Fatalf("Expected 3 tabs, got %d. Contents: \n%s", len(parsed), provider.FileContents)
	}

	if parsed[0].Name != "Tab1" {
		t.Errorf("Expected 0th tab to be Tab1, got %s", parsed[0].Name)
	}
	if parsed[2].Name != "NewTab" {
		t.Errorf("Expected 2nd tab to be NewTab, got %s", parsed[2].Name)
	}
}
