package gobookmarks

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"os"
)

var (
	commitAuthorName  = getenvDefault("GBM_COMMIT_NAME", "Gobookmarks")
	commitAuthorEmail = getenvDefault("GBM_COMMIT_EMAIL", "Gobookmarks@arran.net.au")
	RepoName          = GetBookmarksRepoName()
)

func GetBookmarksRepoName() string {
	ns := os.Getenv("GBM_NAMESPACE")
	if ns != "" {
		return fmt.Sprintf("MyBookmarks-%s", ns)
	}
	return "MyBookmarks"
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	return CurrentProvider.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	return CurrentProvider.CreateBookmarks(ctx, user, token, branch, text)
}

func GetBookmarks(ctx context.Context, user string, ref string, token *oauth2.Token) (string, string, error) {
	return CurrentProvider.GetBookmarks(ctx, user, ref, token)
}
