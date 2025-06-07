package gobookmarks

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	"net/http"
)

var (
	commitAuthor = &github.CommitAuthor{
		Name:  SP("Gobookmarks"),
		Email: SP("Gobookmarks@arran.net.au"),
	}
	RepoName = GetBookmarksRepoName()
)

func GetBookmarksRepoName() string {
	if Namespace != "" {
		return fmt.Sprintf("MyBookmarks-%s", Namespace)
	}
	return "MyBookmarks"
}

func UpdateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	defaultBranch, created, err := GetDefaultBranch(ctx, githubUser, client, branch)
	if err != nil {
		return err
	}
	if branch == "" {
		branch = defaultBranch
	}
	branchRef := "refs/heads/" + branch
	if sourceRef == "" {
		sourceRef = branchRef
	}
	if created {
		return CreateBookmarks(ctx, githubUser, userToken, branch, text)
	}
	_, grefResp, err := client.Git.GetRef(ctx, githubUser, RepoName, branchRef)
	if err != nil && grefResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("UpdateBookmarks: client.Git.GetRef: %w", err)
	}
	if grefResp.StatusCode == http.StatusNotFound {
		err := CreateRef(ctx, githubUser, client, sourceRef, branchRef)
		if err != nil {
			return fmt.Errorf("create ref: %w", err)
		}
	}

	contents, _, resp, err := client.Repositories.GetContents(ctx, githubUser, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: branchRef,
	})
	switch resp.StatusCode {
	case http.StatusNotFound:
		if _, err := CreateRepo(ctx, githubUser, client); err != nil {
			return fmt.Errorf("CreateRepo: %w", err)
		}
		return CreateBookmarks(ctx, githubUser, userToken, branch, text)
	}
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: client.Repositories.GetContents: %w", err)
	}
	if contents == nil || contents.Content == nil {
		return nil
	}
	if expectSHA != "" && contents.SHA != nil && *contents.SHA != expectSHA {
		return fmt.Errorf("bookmarks modified concurrently")
	}
	_, _, err = client.Repositories.UpdateFile(ctx, githubUser, RepoName, "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message: SP("Auto change from web"),
		Content: []byte(text),
		Branch:  &branch,
		SHA:     contents.SHA,
		//SHA:       gref.Object.SHA,
		Author:    commitAuthor,
		Committer: commitAuthor,
	})
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: %w", err)
	}
	return nil
}

func GetDefaultBranch(ctx context.Context, githubUser string, client *github.Client, branch string) (string, bool, error) {
	rep, resp, err := client.Repositories.Get(ctx, githubUser, RepoName)
	created := false
	switch resp.StatusCode {
	case http.StatusNotFound:
		rep, err = CreateRepo(ctx, githubUser, client)
		if err != nil {
			return "", created, err
		}
		created = true
	default:
	}
	if err != nil {
		return "", created, fmt.Errorf("Repositories.Get: %w", err)
	}
	if rep.DefaultBranch != nil {
		branch = *rep.DefaultBranch
	} else {
		branch = "main"
	}
	return branch, created, nil
}

func CreateRepo(ctx context.Context, githubUser string, client *github.Client) (*github.Repository, error) {
	rep := &github.Repository{
		Name:        &RepoName,
		Description: SP("Personal bookmarks"),
		Private:     BP(true),
	}
	var err error
	rep, _, err = client.Repositories.Create(ctx, "", rep)
	if err != nil {
		return nil, fmt.Errorf("Repositories.Create: %w", err)
	}
	if _, _, err = client.Repositories.CreateFile(ctx, githubUser, RepoName, "readme.md", &github.RepositoryContentFileOptions{
		Message: SP("Auto create from web"),
		Content: []byte(`# Your bookmarks 

See . https://github.com/arran4/gobookmarks `),
		Author:    commitAuthor,
		Committer: commitAuthor,
	}); err != nil {
		return nil, fmt.Errorf("CreateReadme: %w", err)
	}

	return rep, err
}

func CreateRef(ctx context.Context, githubUser string, client *github.Client, sourceRef string, branchRef string) error {
	gsref, resp, err := client.Git.GetRef(ctx, githubUser, RepoName, sourceRef)
	switch resp.StatusCode {
	case http.StatusNotFound:
		err = nil
	}
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: client.Git.GetRef sRef: %w", err)
	}
	_, _, err = client.Git.CreateRef(ctx, githubUser, RepoName, &github.Reference{
		Ref:    &branchRef,
		Object: gsref.Object,
	})
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: client.Git.CreateRef sRef: %w", err)
	}
	return nil
}

func SP(s string) *string {
	return &s
}

func BP(b bool) *bool {
	return &b
}

func CreateBookmarks(ctx context.Context, githubUser string, userToken *oauth2.Token, branch, text string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	if branch == "" {
		var err error
		branch, _, err = GetDefaultBranch(ctx, githubUser, client, branch)
		if err != nil {
			return err
		}
	}
	_, _, err := client.Repositories.CreateFile(ctx, githubUser, RepoName, "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto create from web"),
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

func GetBookmarks(ctx context.Context, githubUser string, ref string, userToken *oauth2.Token) (string, string, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(userToken)))
	contents, _, resp, err := client.Repositories.GetContents(ctx, githubUser, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	switch resp.StatusCode {
	case http.StatusNotFound:
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("GetBookmarks: %w", err)
	}
	if contents.Content == nil {
		return "", "", nil
	}
	s, err := base64.StdEncoding.DecodeString(*contents.Content)
	if err != nil {
		return "", "", fmt.Errorf("GetBookmarks: StdEncoding.DecodeString: %w", err)
	}
	sha := ""
	if contents.SHA != nil {
		sha = *contents.SHA
	}
	return string(s), sha, nil
}
