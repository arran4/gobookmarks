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

func GetDefaultBranch(ctx context.Context, githubUser string, client *github.Client, branch string) (string, bool, error) {
	return CurrentProvider.GetDefaultBranch
}

func CreateRepo(ctx context.Context, githubUser string, client *github.Client) (*github.Repository, error) {
	return CurrentProvider.CreateRepo
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	return CurrentProvider.CreateBookmarks(ctx, user, token, branch, text)
}

func CreateRef(ctx context.Context, githubUser string, client *github.Client, sourceRef string, branchRef string) error {
	return CurrentProvider.CreateRef
}

func SP(s string) *string {
	return &s
}

func BP(b bool) *bool {
	return &b
}

func CreateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, branch, text string) error {
	return CurrentProvider.CreateBookmarks
}

