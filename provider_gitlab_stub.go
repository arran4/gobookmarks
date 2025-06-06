//go:build exclude_gitlab

package gobookmarks

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
)

func (GitLabProvider) OAuth2Endpoint() oauth2.Endpoint { return oauth2.Endpoint{} }
func (GitLabProvider) OAuth2Scopes() []string          { return nil }
func (GitLabProvider) GetUserLogin(context.Context, *oauth2.Token) (string, error) {
	return "", fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) GetTags(context.Context, string, *oauth2.Token) ([]*RepositoryTag, error) {
	return nil, fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) GetBranches(context.Context, string, *oauth2.Token) ([]*Branch, error) {
	return nil, fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) GetCommits(context.Context, string, *oauth2.Token) ([]*RepositoryCommit, error) {
	return nil, fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) UpdateBookmarks(context.Context, string, *oauth2.Token, string, string, string, string) error {
	return fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) CreateBookmarks(context.Context, string, *oauth2.Token, string, string) error {
	return fmt.Errorf("gitlab support not built")
}
func (GitLabProvider) GetBookmarks(context.Context, string, string, *oauth2.Token) (string, string, error) {
	return "", "", fmt.Errorf("gitlab support not built")
}
