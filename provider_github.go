//go:build !exclude_github

package gobookmarks

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"net/http"
)

func init() { addCapability("github") }

// OAuth2Endpoint returns the GitHub OAuth2 endpoint.
func (GitHubProvider) OAuth2Endpoint() oauth2.Endpoint { return endpoints.GitHub }

// OAuth2Scopes returns the scopes needed for GitHub.
func (GitHubProvider) OAuth2Scopes() []string { return []string{"repo", "read:user", "user:email"} }

func (GitHubProvider) GetUserLogin(ctx context.Context, token *oauth2.Token) (string, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("client.Users.Get: %w", err)
	}
	if user.Login == nil {
		return "", fmt.Errorf("login not found")
	}
	return *user.Login, nil
}

func (GitHubProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryTag, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	tags, _, err := client.Repositories.ListTags(ctx, user, RepoName, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListTags: %w", err)
	}
	var out []*RepositoryTag
	for _, t := range tags {
		if t.Name == nil {
			continue
		}
		out = append(out, &RepositoryTag{Name: *t.Name})
	}
	return out, nil
}

func (GitHubProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	branches, _, err := client.Repositories.ListBranches(ctx, user, RepoName, &github.BranchListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	var out []*Branch
	for _, b := range branches {
		if b.Name == nil {
			continue
		}
		out = append(out, &Branch{Name: *b.Name})
	}
	return out, nil
}

func (GitHubProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryCommit, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	commits, _, err := client.Repositories.ListCommits(ctx, user, RepoName, &github.CommitsListOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListCommits: %w", err)
	}
	var out []*RepositoryCommit
	for _, c := range commits {
		if c.SHA == nil || c.Commit == nil || c.Commit.Committer == nil {
			continue
		}
		rc := &RepositoryCommit{SHA: *c.SHA}
		rc.Commit.Message = SPV(c.Commit.Message)
		if c.Commit.Committer.Date != nil {
			rc.Commit.Committer.Date = c.Commit.Committer.Date.Time
		}
		rc.Commit.Committer.Name = SPV(c.Commit.Committer.Name)
		rc.Commit.Committer.Email = SPV(c.Commit.Committer.Email)
		out = append(out, rc)
	}
	return out, nil
}

var commitAuthorGitHub = &github.CommitAuthor{
	Name:  SP(commitAuthorName),
	Email: SP(commitAuthorEmail),
}

func (GitHubProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	defaultBranch, created, err := GetDefaultBranch(ctx, user, client, branch)
	if err != nil {
		return fmt.Errorf("GetDefaultBranch: %w", err)
	}
	if branch == "" {
		branch = defaultBranch
	}
	branchRef := "refs/heads/" + branch
	if sourceRef == "" {
		sourceRef = branchRef
	}
	if created {
		return CreateBookmarks(ctx, user, token, branch, text)
	}
	_, grefResp, err := client.Git.GetRef(ctx, user, RepoName, branchRef)
	if err != nil && grefResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("UpdateBookmarks: client.Git.GetRef: %w", err)
	}
	if grefResp.StatusCode == http.StatusNotFound {
		err := CreateRef(ctx, user, client, sourceRef, branchRef)
		if err != nil {
			return fmt.Errorf("create ref: %w", err)
		}
	}

	contents, _, resp, err := client.Repositories.GetContents(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{
		Ref: branchRef,
	})
	switch resp.StatusCode {
	case http.StatusNotFound:
		if _, err := CreateRepo(ctx, user, client); err != nil {
			return fmt.Errorf("CreateRepo: %w", err)
		}
		return CreateBookmarks(ctx, user, token, branch, text)
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
	_, _, err = client.Repositories.UpdateFile(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto change from web"),
		Content:   []byte(text),
		Branch:    &branch,
		SHA:       contents.SHA,
		Author:    commitAuthorGitHub,
		Committer: commitAuthorGitHub,
	})
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: %w", err)
	}
	return nil
}

func (GitHubProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	if branch == "" {
		var err error
		branch, _, err = GetDefaultBranch(ctx, user, client, branch)
		if err != nil {
			return fmt.Errorf("GetDefaultBranch: %w", err)
		}
	}
	_, _, err := client.Repositories.CreateFile(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto create from web"),
		Content:   []byte(text),
		Branch:    &branch,
		Author:    commitAuthorGitHub,
		Committer: commitAuthorGitHub,
	})
	if err != nil {
		return fmt.Errorf("CreateBookmarks: %w", err)
	}
	return nil
}

func (GitHubProvider) GetBookmarks(ctx context.Context, user string, ref string, token *oauth2.Token) (string, string, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	contents, _, resp, err := client.Repositories.GetContents(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{Ref: ref})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
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
		Author:    commitAuthorGitHub,
		Committer: commitAuthorGitHub,
	}); err != nil {
		return nil, fmt.Errorf("CreateReadme: %w", err)
	}
	return rep, err
}

func CreateRef(ctx context.Context, githubUser string, client *github.Client, sourceRef string, branchRef string) error {
	gsref, resp, err := client.Git.GetRef(ctx, githubUser, RepoName, sourceRef)
	if resp.StatusCode == http.StatusNotFound {
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

func SPV(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func SP(s string) *string { return &s }
func BP(b bool) *bool     { return &b }
