//go:build !nogitlab

package gobookmarks

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

// GitLabProvider implements Provider for GitLab.
//
// The GitLab server URL can be overridden using the GitlabServer variable
// defined in settings.go.
type GitLabProvider struct{}

func gitlabUnauthorized(err error) bool {
	var respErr *gitlab.ErrorResponse
	return errors.As(err, &respErr) && respErr.Response != nil && respErr.Response.StatusCode == http.StatusUnauthorized
}

func init() {
	gob.Register(&gitlab.User{})
	RegisterProvider(GitLabProvider{})
}

func (GitLabProvider) Name() string { return "gitlab" }

func (GitLabProvider) DefaultServer() string { return "https://gitlab.com" }

func (GitLabProvider) Config(ctx context.Context, clientID, clientSecret, redirectURL string) *oauth2.Config {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	server := strings.TrimRight(cfg.GitlabServer, "/")
	if server == "" {
		server = "https://gitlab.com"
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"api"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  server + "/oauth/authorize",
			TokenURL: server + "/oauth/token",
		},
	}
}

func (GitLabProvider) client(ctx context.Context, token *oauth2.Token) (*gitlab.Client, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	server := cfg.GitlabServer
	if server == "" {
		server = "https://gitlab.com"
	}
	return gitlab.NewOAuthClient(token.AccessToken, gitlab.WithBaseURL(server))
}

func (GitLabProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab CurrentUser client: %v", err)
		return nil, err
	}
	u, _, err := c.Users.CurrentUser()
	if err != nil {
		log.Printf("gitlab CurrentUser lookup: %v", err)
		return nil, err
	}
	return &User{Login: u.Username}, nil
}

func (GitLabProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab GetTags client: %v", err)
		return nil, err
	}
	tags, _, err := c.Tags.ListTags(user+"/"+repoName, &gitlab.ListTagsOptions{})
	if err != nil {
		if gitlabUnauthorized(err) {
			return nil, ErrSignedOut
		}
		log.Printf("gitlab GetTags: %v", err)
		return nil, fmt.Errorf("ListTags: %w", err)
	}
	res := make([]*Tag, 0, len(tags))
	for _, t := range tags {
		res = append(res, &Tag{Name: t.Name})
	}
	return res, nil
}

func (GitLabProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab GetBranches client: %v", err)
		return nil, err
	}
	bs, _, err := c.Branches.ListBranches(user+"/"+repoName, &gitlab.ListBranchesOptions{})
	if err != nil {
		if gitlabUnauthorized(err) {
			return nil, ErrSignedOut
		}
		log.Printf("gitlab GetBranches: %v", err)
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	res := make([]*Branch, 0, len(bs))
	for _, b := range bs {
		res = append(res, &Branch{Name: b.Name})
	}
	return res, nil
}

func (GitLabProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab GetCommits client: %v", err)
		return nil, err
	}
	cs, _, err := c.Commits.ListCommits(user+"/"+repoName, &gitlab.ListCommitsOptions{RefName: &ref, ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage}})
	if err != nil {
		if gitlabUnauthorized(err) {
			return nil, ErrSignedOut
		}
		log.Printf("gitlab GetCommits: %v", err)
		return nil, fmt.Errorf("ListCommits: %w", err)
	}
	res := make([]*Commit, 0, len(cs))
	for _, commit := range cs {
		res = append(res, &Commit{
			SHA:            commit.ID,
			Message:        commit.Message,
			CommitterName:  commit.CommitterName,
			CommitterEmail: commit.CommitterEmail,
			CommitterDate:  *commit.CommittedDate,
		})
	}
	return res, nil
}

func (GitLabProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab GetBookmarks client: %v", err)
		return "", "", err
	}
	if ref == "" {
		ref = "HEAD"
	}
	f, _, err := c.RepositoryFiles.GetFile(user+"/"+repoName, "bookmarks.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr(ref)})
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return "", "", nil
		}
		if respErr, ok := err.(*gitlab.ErrorResponse); ok {
			if respErr.Response != nil && respErr.Response.StatusCode == http.StatusNotFound {
				return "", "", nil
			}
			if gitlabUnauthorized(err) {
				return "", "", ErrSignedOut
			}
			log.Printf("gitlab GetBookmarks get file: %v", err)
			return "", "", nil
		}
		if gitlabUnauthorized(err) {
			return "", "", ErrSignedOut
		}
		log.Printf("gitlab GetBookmarks: %v", err)
		return "", "", err
	}
	data, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		log.Printf("gitlab GetBookmarks decode: %v", err)
		return "", "", err
	}
	return string(data), f.LastCommitID, nil
}

func (GitLabProvider) getDefaultBranch(ctx context.Context, user string, client *gitlab.Client, branch string) (string, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	p, _, err := client.Projects.GetProject(user+"/"+repoName, nil)
	if err != nil {
		if respErr, ok := err.(*gitlab.ErrorResponse); ok {
			if respErr.Response != nil && respErr.Response.StatusCode == http.StatusNotFound {
				return "", ErrRepoNotFound
			}
			if gitlabUnauthorized(err) {
				return "", ErrSignedOut
			}
		}
		if gitlabUnauthorized(err) {
			return "", ErrSignedOut
		}
		log.Printf("gitlab getDefaultBranch: %v", err)
		return "", err
	}
	if p.DefaultBranch != "" {
		branch = p.DefaultBranch
	} else {
		branch = "main"
	}
	return branch, nil
}
func (GitLabProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab UpdateBookmarks client: %v", err)
		return err
	}
	if branch == "" {
		branch, err = GitLabProvider{}.getDefaultBranch(ctx, user, c, branch)
		if err != nil {
			log.Printf("gitlab UpdateBookmarks default branch: %v", err)
			return err
		}
	}
	opt := &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(branch),
		Content:       gitlab.Ptr(text),
		AuthorEmail:   gitlab.Ptr("Gobookmarks@arran.net.au"),
		AuthorName:    gitlab.Ptr("Gobookmarks"),
		LastCommitID:  gitlab.Ptr(expectSHA),
		CommitMessage: gitlab.Ptr("Auto change from web"),
	}
	_, _, err = c.RepositoryFiles.UpdateFile(user+"/"+repoName, "bookmarks.txt", opt)
	if err != nil {
		var respErr *gitlab.ErrorResponse
		if errors.As(err, &respErr) {
			if respErr.Response != nil && respErr.Response.StatusCode == http.StatusNotFound {
				return ErrRepoNotFound
			}
			if gitlabUnauthorized(err) {
				return ErrSignedOut
			}
			log.Printf("gitlab UpdateBookmarks update file: %v", err)
			return err
		}
		if gitlabUnauthorized(err) {
			return ErrSignedOut
		}
		if err.Error() == "404 Not Found" {
			return ErrRepoNotFound
		}
		log.Printf("gitlab UpdateBookmarks: %v", err)
		return err
	}
	return nil
}

func (GitLabProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		log.Printf("gitlab CreateBookmarks client: %v", err)
		return err
	}
	if branch == "" {
		branch, err = GitLabProvider{}.getDefaultBranch(ctx, user, c, branch)
		if err != nil {
			log.Printf("gitlab CreateBookmarks default branch: %v", err)
			return err
		}
	}
	opt := &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr(branch),
		Content:       gitlab.Ptr(text),
		AuthorEmail:   gitlab.Ptr("Gobookmarks@arran.net.au"),
		AuthorName:    gitlab.Ptr("Gobookmarks"),
		CommitMessage: gitlab.Ptr("Auto create from web"),
	}
	_, _, err = c.RepositoryFiles.CreateFile(user+"/"+repoName, "bookmarks.txt", opt)
	if err != nil {
		if respErr, ok := err.(*gitlab.ErrorResponse); ok {
			if respErr.Response != nil && respErr.Response.StatusCode == http.StatusNotFound {
				return ErrRepoNotFound
			}
			if gitlabUnauthorized(err) {
				return ErrSignedOut
			}
			log.Printf("gitlab CreateBookmarks create file: %v", err)
			return err
		}
		if gitlabUnauthorized(err) {
			return ErrSignedOut
		}
		log.Printf("gitlab CreateBookmarks: %v", err)
		return err
	}
	return nil
}

func (p GitLabProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		return err
	}
	// RepoName = name
	_, _, err = c.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name:                 gitlab.Ptr(name),
		Description:          gitlab.Ptr("Personal bookmarks"),
		Visibility:           gitlab.Ptr(gitlab.PrivateVisibility),
		InitializeWithReadme: gitlab.Ptr(true),
	})
	if err != nil {
		if respErr, ok := err.(*gitlab.ErrorResponse); ok && respErr.Response != nil &&
			(respErr.Response.StatusCode == http.StatusBadRequest || respErr.Response.StatusCode == http.StatusConflict) {
			// repository already exists
			err = nil
		} else if gitlabUnauthorized(err) {
			return ErrSignedOut
		}
	}
	return err
}

func (p GitLabProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	c, err := GitLabProvider{}.client(ctx, token)
	if err != nil {
		return false, err
	}
	_, resp, err := c.Projects.GetProject(user+"/"+name, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		if gitlabUnauthorized(err) {
			return false, ErrSignedOut
		}
		return false, err
	}
	return true, nil
}
