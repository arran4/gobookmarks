//go:build !nogithub

package gobookmarks

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

// GitHubProvider implements Provider for GitHub.
type GitHubProvider struct{}

func init() { RegisterProvider(GitHubProvider{}) }

func (GitHubProvider) Name() string { return "github" }

func (GitHubProvider) DefaultServer() string { return "https://github.com" }

func (GitHubProvider) OAuth2Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	server := strings.TrimRight(GitServer, "/")
	if server == "" {
		server = "https://github.com"
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"repo", "read:user", "user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  server + "/login/oauth/authorize",
			TokenURL: server + "/login/oauth/access_token",
		},
	}
}

func (GitHubProvider) client(ctx context.Context, token *oauth2.Token) *github.Client {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	server := strings.TrimRight(GitServer, "/")
	if server == "" || server == "https://github.com" {
		return github.NewClient(httpClient)
	}
	c, err := github.NewEnterpriseClient(server+"/api/v3/", server+"/upload/v3/", httpClient)
	if err != nil {
		return github.NewClient(httpClient)
	}
	return c
}

func (p GitHubProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	u, _, err := p.client(ctx, token).Users.Get(ctx, "")
	if err != nil {
		log.Printf("github CurrentUser: %v", err)
		return nil, err
	}
	user := &User{}
	if u.Login != nil {
		user.Login = *u.Login
	}
	return user, nil
}

func (p GitHubProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	tags, _, err := p.client(ctx, token).Repositories.ListTags(ctx, user, RepoName, &github.ListOptions{})
	if err != nil {
		log.Printf("github GetTags: %v", err)
		return nil, fmt.Errorf("ListTags: %w", err)
	}
	res := make([]*Tag, 0, len(tags))
	for _, t := range tags {
		res = append(res, &Tag{Name: t.GetName()})
	}
	return res, nil
}

func (p GitHubProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	bs, _, err := p.client(ctx, token).Repositories.ListBranches(ctx, user, RepoName, &github.BranchListOptions{})
	if err != nil {
		log.Printf("github GetBranches: %v", err)
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	res := make([]*Branch, 0, len(bs))
	for _, b := range bs {
		res = append(res, &Branch{Name: b.GetName()})
	}
	return res, nil
}

func (p GitHubProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	cs, _, err := p.client(ctx, token).Repositories.ListCommits(ctx, user, RepoName, &github.CommitsListOptions{})
	if err != nil {
		log.Printf("github GetCommits: %v", err)
		return nil, fmt.Errorf("ListCommits: %w", err)
	}
	res := make([]*Commit, 0, len(cs))
	for _, c := range cs {
		cm := c.GetCommit()
		com := &Commit{SHA: c.GetSHA()}
		if cm != nil {
			com.Message = cm.GetMessage()
			comm := cm.Committer
			if comm != nil {
				com.CommitterName = comm.GetName()
				com.CommitterEmail = comm.GetEmail()
				com.CommitterDate = comm.GetDate().Time
			}
		}
		res = append(res, com)
	}
	return res, nil
}

func (p GitHubProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	contents, _, resp, err := p.client(ctx, token).Repositories.GetContents(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{Ref: ref})
	if resp != nil && resp.StatusCode == 404 {
		return "", "", nil
	}
	if err != nil {
		log.Printf("github GetBookmarks: %v", err)
		return "", "", fmt.Errorf("GetBookmarks: %w", err)
	}
	if contents.Content == nil {
		return "", "", nil
	}
	b, err := base64.StdEncoding.DecodeString(*contents.Content)
	if err != nil {
		log.Printf("github GetBookmarks decode: %v", err)
		return "", "", fmt.Errorf("GetBookmarks: %w", err)
	}
	sha := ""
	if contents.SHA != nil {
		sha = *contents.SHA
	}
	return string(b), sha, nil
}

var commitAuthor = &github.CommitAuthor{Name: SP("Gobookmarks"), Email: SP("Gobookmarks@arran.net.au")}

func (p GitHubProvider) getDefaultBranch(ctx context.Context, user string, client *github.Client, branch string) (string, error) {
	rep, resp, err := client.Repositories.Get(ctx, user, RepoName)
	if resp != nil && resp.StatusCode == 404 {
		return "", ErrRepoNotFound
	}
	if err != nil {
		log.Printf("github getDefaultBranch: %v", err)
		return "", fmt.Errorf("Repositories.Get: %w", err)
	}
	if rep.DefaultBranch != nil {
		branch = *rep.DefaultBranch
	} else {
		branch = "main"
	}
	return branch, nil
}

func (p GitHubProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	client := p.client(ctx, token)
	RepoName = name
	rep := &github.Repository{Name: &RepoName, Description: SP("Personal bookmarks"), Private: BP(true)}
	rep, _, err := client.Repositories.Create(ctx, "", rep)
	if err != nil {
		log.Printf("github createRepo: %v", err)
		return fmt.Errorf("Repositories.Create: %w", err)
	}
	_, _, err = client.Repositories.CreateFile(ctx, user, RepoName, "readme.md", &github.RepositoryContentFileOptions{
		Message: SP("Auto create from web"),
		Content: []byte(`# Your bookmarks

See . https://github.com/arran4/gobookmarks `),
		Author: commitAuthor, Committer: commitAuthor,
	})
	if err != nil {
		log.Printf("github createRepo readme: %v", err)
		return fmt.Errorf("CreateReadme: %w", err)
	}
	_ = rep
	return nil
}

func (p GitHubProvider) createRef(ctx context.Context, user string, client *github.Client, sourceRef, branchRef string) error {
	gsref, resp, err := client.Git.GetRef(ctx, user, RepoName, sourceRef)
	if resp != nil && resp.StatusCode == 404 {
		err = nil
	}
	if err != nil {
		log.Printf("github createRef getRef: %v", err)
		return fmt.Errorf("GetRef: %w", err)
	}
	_, _, err = client.Git.CreateRef(ctx, user, RepoName, &github.Reference{Ref: &branchRef, Object: gsref.Object})
	if err != nil {
		log.Printf("github createRef create: %v", err)
		return fmt.Errorf("CreateRef: %w", err)
	}
	return nil
}

func (p GitHubProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	client := p.client(ctx, token)
	defaultBranch, err := p.getDefaultBranch(ctx, user, client, branch)
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
	_, grefResp, err := client.Git.GetRef(ctx, user, RepoName, branchRef)
	if err != nil && grefResp.StatusCode != 404 {
		log.Printf("github UpdateBookmarks getRef: %v", err)
		return fmt.Errorf("GetRef: %w", err)
	}
	if grefResp.StatusCode == 404 {
		if err := p.createRef(ctx, user, client, sourceRef, branchRef); err != nil {
			log.Printf("github UpdateBookmarks create ref: %v", err)
			return fmt.Errorf("create ref: %w", err)
		}
	}
	contents, _, resp, err := client.Repositories.GetContents(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentGetOptions{Ref: branchRef})
	if resp != nil && resp.StatusCode == 404 {
		return ErrRepoNotFound
	}
	if err != nil {
		log.Printf("github UpdateBookmarks get contents: %v", err)
		return fmt.Errorf("GetContents: %w", err)
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
		Author:    commitAuthor,
		Committer: commitAuthor,
	})
	if err != nil {
		log.Printf("github UpdateBookmarks update: %v", err)
		return fmt.Errorf("UpdateBookmarks: %w", err)
	}
	return nil
}

func (p GitHubProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	client := p.client(ctx, token)
	if branch == "" {
		var err error
		branch, err = p.getDefaultBranch(ctx, user, client, branch)
		if err != nil {
			log.Printf("github CreateBookmarks default branch: %v", err)
			return err
		}
	}
	_, _, err := client.Repositories.CreateFile(ctx, user, RepoName, "bookmarks.txt", &github.RepositoryContentFileOptions{
		Message:   SP("Auto create from web"),
		Content:   []byte(text),
		Branch:    &branch,
		Author:    commitAuthor,
		Committer: commitAuthor,
	})
	if err != nil {
		log.Printf("github CreateBookmarks: %v", err)
		return fmt.Errorf("CreateBookmarks: %w", err)
	}
	return nil
}
