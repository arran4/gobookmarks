//go:build exclude_github

package gobookmarks

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
)

func (GitHubProvider) OAuth2Endpoint() oauth2.Endpoint { return oauth2.Endpoint{} }
func (GitHubProvider) OAuth2Scopes() []string          { return nil }
func (GitHubProvider) GetUserLogin(context.Context, *oauth2.Token) (string, error) {
	return "", fmt.Errorf("github support not built")
}
func (GitHubProvider) GetTags(context.Context, string, *oauth2.Token) ([]*RepositoryTag, error) {
	return nil, fmt.Errorf("github support not built")
}
func (GitHubProvider) GetBranches(context.Context, string, *oauth2.Token) ([]*Branch, error) {
	return nil, fmt.Errorf("github support not built")
}
func (GitHubProvider) GetCommits(context.Context, string, *oauth2.Token) ([]*RepositoryCommit, error) {
	return nil, fmt.Errorf("github support not built")
}
func (GitHubProvider) UpdateBookmarks(context.Context, string, *oauth2.Token, string, string, string) error {
	return fmt.Errorf("github support not built")
}
func (GitHubProvider) CreateBookmarks(context.Context, string, *oauth2.Token, string, string) error {
	return fmt.Errorf("github support not built")
}
func (GitHubProvider) GetBookmarks(context.Context, string, string, *oauth2.Token) (string, error) {
	return "", fmt.Errorf("github support not built")
}
