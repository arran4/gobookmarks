package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
)

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryTag, error) {
	return CurrentProvider.GetTags(ctx, user, token)
}

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	return CurrentProvider.GetBranches(ctx, user, token)
}
func GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryCommit, error) {
	return CurrentProvider.GetCommits(ctx, user, token)
}
