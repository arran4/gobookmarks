//go:build !nogithub

package gobookmarks

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// GitHubProvider implements Provider for GitHub.
type GitHubProvider struct{}

func init() { RegisterProvider(GitHubProvider{}) }

func (GitHubProvider) Name() string { return "github" }

func (GitHubProvider) OAuth2Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"repo", "read:user", "user:email"},
		Endpoint:     endpoints.GitHub,
	}
}

func (GitHubProvider) client(ctx context.Context, token *oauth2.Token) *github.Client {
	return github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
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

func (p GitHubProvider) getDefaultBranch(ctx context.Context, user string, client *github.Client, branch string) (string, bool, error) {
	rep, resp, err := client.Repositories.Get(ctx, user, RepoName)
	created := false
	if resp != nil && resp.StatusCode == 404 {
		rep, err = p.createRepo(ctx, user, client)
		if err != nil {
			log.Printf("github createRepo: %v", err)
			return "", created, err
		}
		created = true
	}
	if err != nil {
		log.Printf("github getDefaultBranch: %v", err)
		return "", created, fmt.Errorf("Repositories.Get: %w", err)
	}
	if rep.DefaultBranch != nil {
		branch = *rep.DefaultBranch
	} else {
		branch = "main"
	}
	return branch, created, nil
}

func (p GitHubProvider) createRepo(ctx context.Context, user string, client *github.Client) (*github.Repository, error) {
	rep := &github.Repository{Name: &RepoName, Description: SP("Personal bookmarks"), Private: BP(true)}
	rep, _, err := client.Repositories.Create(ctx, "", rep)
	if err != nil {
		log.Printf("github createRepo: %v", err)
		return nil, fmt.Errorf("Repositories.Create: %w", err)
	}
	_, _, err = client.Repositories.CreateFile(ctx, user, RepoName, "readme.md", &github.RepositoryContentFileOptions{
		Message: SP("Auto create from web"),
		Content: []byte(`# Your bookmarks

See . https://github.com/arran4/gobookmarks `),
		Author: commitAuthor, Committer: commitAuthor,
	})
	if err != nil {
		log.Printf("github createRepo readme: %v", err)
		return nil, fmt.Errorf("CreateReadme: %w", err)
	}
	return rep, nil
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
	defaultBranch, created, err := p.getDefaultBranch(ctx, user, client, branch)
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
		return p.CreateBookmarks(ctx, user, token, branch, text)
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
		if _, err := p.createRepo(ctx, user, client); err != nil {
			log.Printf("github UpdateBookmarks create repo: %v", err)
			return fmt.Errorf("CreateRepo: %w", err)
		}
		return p.CreateBookmarks(ctx, user, token, branch, text)
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
		branch, _, err = p.getDefaultBranch(ctx, user, client, branch)
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
