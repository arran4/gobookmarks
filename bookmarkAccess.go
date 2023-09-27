package a4webbm

import (
	"context"
	"fmt"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

func UpdateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, text string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	_, _, err := client.Repositories.UpdateFile(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message: SP("Auto change from web"),
		Content: []byte(text),
		Branch:  SP("main"),
		Author: &github.CommitAuthor{
			Name:  SP("Gobookmarks"),
			Email: SP("Gobookmarks@arran.net.au"),
		},
		Committer: nil,
	})
	if err != nil {
		return fmt.Errorf("CreateBookmarks: %w", err)
	}
	return nil
}

func SP(s string) *string {
	return &s
}

func CreateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, text string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	_, _, err := client.Repositories.CreateFile(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message: SP("Auto change from web"),
		Content: []byte(text),
		Branch:  SP("main"),
		Author: &github.CommitAuthor{
			Name:  SP("Gobookmarks"),
			Email: SP("Gobookmarks@arran.net.au"),
		},
		Committer: nil,
	})
	if err != nil {
		return fmt.Errorf("CreateBookmarks: %w", err)
	}
	return nil
}

func GetBookmarksForUser(ctx context.Context, githubUser string, ref string, userToken *oauth2.Token) (string, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	contents, _, _, err := client.Repositories.GetContents(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return "", fmt.Errorf("GetBookmarksForUser: %w", err)
	}
	if contents.Content == nil {
		return "", nil
	}
	return *contents.Content, nil
}
