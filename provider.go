package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
)

// Provider defines interface for git backends.
type Provider interface {
	OAuth2Endpoint() oauth2.Endpoint
	OAuth2Scopes() []string
	GetUserLogin(ctx context.Context, token *oauth2.Token) (string, error)
	GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryTag, error)
	GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryCommit, error)
	UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	GetBookmarks(ctx context.Context, user string, ref string, token *oauth2.Token) (string, string, error)
}

// GitHubProvider implements the GitHub backend.
type GitHubProvider struct{}

// GitLabProvider implements the GitLab backend.
type GitLabProvider struct{}

var (
	// CurrentProvider holds the active backend implementation.
	CurrentProvider Provider
	// GitLabBaseURL is the base API URL for GitLab.
	GitLabBaseURL string
	// capabilities lists providers compiled into the binary.
	capabilities []string
)

func addCapability(c string) {
	capabilities = append(capabilities, c)
}

// Capabilities returns the compiled provider capabilities.
func Capabilities() []string { return append([]string(nil), capabilities...) }
