//go:build !exclude_gitlab

package gobookmarks

import (
	"context"
	"encoding/base64"
	"fmt"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"
	"strings"
)

func init() { addCapability("gitlab") }

// OAuth2Endpoint returns the GitLab OAuth2 endpoint.
func (GitLabProvider) OAuth2Endpoint() oauth2.Endpoint {
	base := strings.TrimSuffix(GitLabBaseURL, "/api/v4")
	return oauth2.Endpoint{
		AuthURL:  base + "/oauth/authorize",
		TokenURL: base + "/oauth/token",
	}
}

// OAuth2Scopes returns the scopes needed for GitLab.
func (GitLabProvider) OAuth2Scopes() []string { return []string{"api", "read_user"} }

func (GitLabProvider) GetUserLogin(ctx context.Context, token *oauth2.Token) (string, error) {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get client: %w", err)
	}
	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return "", fmt.Errorf("user error: %w", err)
	}
	return user.Username, nil
}

func (GitLabProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryTag, error) {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)
	tags, _, err := client.Tags.ListTags(project, &gitlab.ListTagsOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListTags: %w", err)
	}
	var out []*RepositoryTag
	for _, t := range tags {
		out = append(out, &RepositoryTag{Name: t.Name})
	}
	return out, nil
}

func (GitLabProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)
	branches, _, err := client.Branches.ListBranches(project, &gitlab.ListBranchesOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	var out []*Branch
	for _, b := range branches {
		out = append(out, &Branch{Name: b.Name})
	}
	return out, nil
}

func (GitLabProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*RepositoryCommit, error) {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)
	commits, _, err := client.Commits.ListCommits(project, &gitlab.ListCommitsOptions{})
	if err != nil {
		return nil, fmt.Errorf("ListCommits: %w", err)
	}
	var out []*RepositoryCommit
	for _, c := range commits {
		rc := &RepositoryCommit{SHA: c.ID}
		rc.Commit.Message = c.Message
		if c.CommittedDate != nil {
			rc.Commit.Committer.Date = *c.CommittedDate
		}
		rc.Commit.Committer.Name = c.CommitterName
		rc.Commit.Committer.Email = c.CommitterEmail
		out = append(out, rc)
	}
	return out, nil
}

func (GitLabProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text string) error {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)

	if err := ensureGitLabProject(client, project); err != nil {
		return fmt.Errorf("ensure project: %w", err)
	}

	if branch == "" {
		branch = "main"
	}
	if sourceRef == "" {
		sourceRef = branch
	}

	if err := ensureGitLabBranch(client, project, branch, sourceRef); err != nil {
		return fmt.Errorf("ensure branch: %w", err)
	}

	return writeGitLabBookmarks(client, project, branch, text)
}

func (GitLabProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	return updateGitLabBookmarks(ctx, user, token, "", branch, text)
}

func (GitLabProvider) GetBookmarks(ctx context.Context, user string, ref string, token *oauth2.Token) (string, error) {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return "", err
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)
	file, _, err := client.RepositoryFiles.GetFile(project, "bookmarks.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr(ref)})
	if err != nil {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("decode file: %w", err)
	}
	return string(data), nil
}

func getGitLabClient(ctx context.Context, token *oauth2.Token) (*gitlab.Client, error) {
	opts := []gitlab.ClientOptionFunc{}
	if GitLabBaseURL != "" {
		opts = append(opts, gitlab.WithBaseURL(GitLabBaseURL))
	}
	c, err := gitlab.NewOAuthClient(token.AccessToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("NewOAuthClient: %w", err)
	}
	return c, nil
}

func updateGitLabBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text string) error {
	client, err := getGitLabClient(ctx, token)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}
	project := fmt.Sprintf("%s/%s", user, RepoName)
	if err := ensureGitLabProject(client, project); err != nil {
		return fmt.Errorf("ensure project: %w", err)
	}
	if branch == "" {
		branch = "main"
	}
	if sourceRef == "" {
		sourceRef = branch
	}
	if err := ensureGitLabBranch(client, project, branch, sourceRef); err != nil {
		return fmt.Errorf("ensure branch: %w", err)
	}
	return writeGitLabBookmarks(client, project, branch, text)
}

func ensureGitLabProject(client *gitlab.Client, project string) error {
	_, _, err := client.Projects.GetProject(project, nil)
	if err == nil {
		return nil
	}
	_, _, err = client.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name:       gitlab.Ptr(RepoName),
		Visibility: gitlab.Ptr(gitlab.PrivateVisibility),
	})
	if err != nil {
		return fmt.Errorf("CreateProject: %w", err)
	}
	return nil
}

func ensureGitLabBranch(client *gitlab.Client, project, branch, sourceRef string) error {
	_, _, err := client.Branches.GetBranch(project, branch)
	if err == nil {
		return nil
	}
	_, _, err = client.Branches.CreateBranch(project, &gitlab.CreateBranchOptions{
		Branch: gitlab.Ptr(branch),
		Ref:    gitlab.Ptr(sourceRef),
	})
	if err != nil {
		return fmt.Errorf("CreateBranch: %w", err)
	}
	return nil
}

func writeGitLabBookmarks(client *gitlab.Client, project, branch, text string) error {
	_, _, err := client.RepositoryFiles.GetFile(project, "bookmarks.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr(branch)})
	if err != nil {
		_, _, err = client.RepositoryFiles.CreateFile(project, "bookmarks.txt", &gitlab.CreateFileOptions{
			Branch:        gitlab.Ptr(branch),
			Content:       gitlab.Ptr(text),
			CommitMessage: gitlab.Ptr("Auto create from web"),
			AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
			AuthorName:    gitlab.Ptr(commitAuthorName),
		})
	} else {
		_, _, err = client.RepositoryFiles.UpdateFile(project, "bookmarks.txt", &gitlab.UpdateFileOptions{
			Branch:        gitlab.Ptr(branch),
			Content:       gitlab.Ptr(text),
			CommitMessage: gitlab.Ptr("Auto change from web"),
			AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
			AuthorName:    gitlab.Ptr(commitAuthorName),
		})
	}
	if err != nil {
		return fmt.Errorf("UpdateBookmarks: %w", err)
	}
	return nil
}
