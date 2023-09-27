package gobookmarks

import (
	"context"
	"fmt"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

func GetTags(ctx context.Context, githubUser string, userToken *oauth2.Token) ([]*github.RepositoryTag, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	tags, _, err := client.Repositories.ListTags(ctx, githubUser, "MyBookmarks", &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListTags: %w", err)
	}
	return tags, nil
}

func GetBranches(ctx context.Context, githubUser string, userToken *oauth2.Token) ([]*github.Branch, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	branches, _, err := client.Repositories.ListBranches(ctx, githubUser, "MyBookmarks", &github.BranchListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	return branches, nil
}
func GetCommits(ctx context.Context, githubUser string, userToken *oauth2.Token) ([]*github.RepositoryCommit, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	commits, _, err := client.Repositories.ListCommits(ctx, githubUser, "MyBookmarks", &github.CommitsListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListCommits: %w", err)
	}
	return commits, nil
}
