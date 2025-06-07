package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
)

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	return ActiveProvider.GetTags(ctx, user, token)
}

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	return ActiveProvider.GetBranches(ctx, user, token)
}

func GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	return ActiveProvider.GetCommits(ctx, user, token)
}

func GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	return ActiveProvider.GetBookmarks(ctx, user, ref, token)
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	return ActiveProvider.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	return ActiveProvider.CreateBookmarks(ctx, user, token, branch, text)
}
