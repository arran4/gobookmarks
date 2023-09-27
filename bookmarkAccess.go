package a4webbm

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

var (
	commitAuthor = &github.CommitAuthor{
		Name:  SP("Gobookmarks"),
		Email: SP("Gobookmarks@arran.net.au"),
	}
)

func UpdateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	contents, _, _, err := client.Repositories.GetContents(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: fmt.Sprintf("refs/heads/%s", branch),
	})
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: client.Repositories.GetContents: %w", err)
	}
	if contents.Content == nil {
		return nil
	}
	_, _, err = client.Repositories.UpdateFile(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto change from web"),
		Content:   []byte(text),
		Branch:    SP("main"),
		SHA:       contents.SHA,
		Author:    commitAuthor,
		Committer: commitAuthor,
	})
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: %w", err)
	}
	return nil
}

func SP(s string) *string {
	return &s
}

func CreateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	_, _, err := client.Repositories.CreateFile(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto change from web"),
		Content:   []byte(text),
		Branch:    &branch,
		Author:    commitAuthor,
		Committer: commitAuthor,
	})
	if err != nil {
		return fmt.Errorf("CreateBookmarks: %w", err)
	}
	return nil
}

func GetBookmarks(ctx context.Context, githubUser string, ref string, userToken *oauth2.Token) (string, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	contents, _, _, err := client.Repositories.GetContents(ctx, githubUser, "MyBookmarks", "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return "", fmt.Errorf("GetBookmarks: %w", err)
	}
	if contents.Content == nil {
		return "", nil
	}
	s, err := base64.StdEncoding.DecodeString(*contents.Content)
	if err != nil {
		return "", fmt.Errorf("GetBookmarks: StdEncoding.DecodeString: %w", err)
	}
	return string(s), nil
}
