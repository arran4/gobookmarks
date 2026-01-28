//go:build !excludegitprovider

package gobookmarks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"golang.org/x/oauth2"
)

// GitProvider implements Provider using local git repositories.
type GitProvider struct{}

func init() { RegisterProvider(GitProvider{}) }

func (GitProvider) Name() string                                                     { return "git" }
func (GitProvider) DefaultServer() string                                            { return "" }
func (GitProvider) Config(ctx context.Context, clientID, clientSecret, redirectURL string) *oauth2.Config {
	return nil
}
func (GitProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return &User{Login: "local"}, nil
}

func userDir(user string, path string) string {
	h := sha256.Sum256([]byte(user))
	return filepath.Join(path, hex.EncodeToString(h[:]))
}

func openRepo(user string, path string) (*git.Repository, error) {
	r, err := git.PlainOpen(userDir(user, path))
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return r, nil
}

func (GitProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	r, err := openRepo(user, cfg.GetLocalGitPath())
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, nil
		}
		return nil, err
	}
	iter, err := r.Tags()
	if err != nil {
		return nil, err
	}
	var tags []*Tag
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, &Tag{Name: ref.Name().Short()})
		return nil
	})
	return tags, err
}

func (GitProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	r, err := openRepo(user, cfg.GetLocalGitPath())
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return []*Branch{{Name: "main"}}, nil
		}
		return nil, err
	}
	iter, err := r.Branches()
	if err != nil {
		return nil, err
	}
	var branches []*Branch
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, &Branch{Name: ref.Name().Short()})
		return nil
	})
	if len(branches) == 0 {
		branches = append(branches, &Branch{Name: "main"})
	}
	return branches, err
}

func (GitProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	r, err := openRepo(user, cfg.GetLocalGitPath())
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if ref == "" {
		ref = "refs/heads/main"
	}
	h, err := r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, nil
	}
	iter, err := r.Log(&git.LogOptions{From: *h})
	if err != nil {
		return nil, err
	}
	start := (page - 1) * perPage
	i := 0
	var commits []*Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if i < start {
			i++
			return nil
		}
		if len(commits) >= perPage {
			return storer.ErrStop
		}
		commits = append(commits, &Commit{
			SHA:            c.Hash.String(),
			Message:        c.Message,
			CommitterName:  c.Committer.Name,
			CommitterEmail: c.Committer.Email,
			CommitterDate:  c.Committer.When,
		})
		i++
		return nil
	})
	if err == storer.ErrStop {
		err = nil
	}
	return commits, err
}

func (GitProvider) AdjacentCommits(ctx context.Context, user string, token *oauth2.Token, ref, sha string) (string, string, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	r, err := openRepo(user, cfg.GetLocalGitPath())
	if err != nil {
		return "", "", err
	}
	if ref == "" {
		ref = "refs/heads/main"
	}
	h, err := r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", "", err
	}
	iter, err := r.Log(&git.LogOptions{From: *h})
	if err != nil {
		return "", "", err
	}
	var prev, next string
	var last string
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == sha {
			// previous commit in history (older)
			if c.NumParents() > 0 {
				p, err := c.Parent(0)
				if err == nil {
					prev = p.Hash.String()
				}
			}
			next = last
			return storer.ErrStop
		}
		last = c.Hash.String()
		return nil
	})
	if err == storer.ErrStop {
		err = nil
	}
	return prev, next, err
}

func (GitProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	r, err := openRepo(user, cfg.GetLocalGitPath())
	if err != nil {
		return "", "", err
	}
	if ref == "" {
		ref = "refs/heads/main"
	}
	h, err := r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", "", ErrRepoNotFound
		}
		return "", "", err
	}
	commit, err := r.CommitObject(*h)
	if err != nil {
		return "", "", err
	}
	file, err := commit.File("bookmarks.txt")
	if err != nil {
		if err == object.ErrFileNotFound {
			return "", commit.Hash.String(), nil
		}
		return "", "", err
	}
	data, err := file.Contents()
	if err != nil {
		return "", "", err
	}
	return data, commit.Hash.String(), nil
}

func (GitProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	if branch == "" {
		branch = "main"
	}
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	path := cfg.GetLocalGitPath()
	r, err := openRepo(user, path)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Keep: true})
	if err != nil {
		return ErrRepoNotFound
	}
	head, err := r.Head()
	if err != nil {
		return err
	}
	if expectSHA != "" && head.Hash().String() != expectSHA {
		return errors.New("sha mismatch")
	}
	if err := os.WriteFile(filepath.Join(userDir(user, path), "bookmarks.txt"), []byte(text), 0600); err != nil {
		return err
	}
	if _, err := wt.Add("bookmarks.txt"); err != nil {
		return err
	}
	_, err = wt.Commit("Auto change from web", &git.CommitOptions{
		Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
	})
	if err != nil && !errors.Is(err, git.ErrEmptyCommit) {
		return err
	}
	return nil
}

func (GitProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	path := cfg.GetLocalGitPath()
	r, err := openRepo(user, path)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Create: true, Keep: true})
	if err != nil {
		if !errors.Is(err, git.ErrBranchExists) {
			// branch already exists
			if err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Keep: true}); err != nil {
				return err
			}
		}
	}
	if err := os.WriteFile(filepath.Join(userDir(user, path), "bookmarks.txt"), []byte(text), 0600); err != nil {
		return err
	}
	if _, err := wt.Add("bookmarks.txt"); err != nil {
		return err
	}
	_, err = wt.Commit("Auto create from web", &git.CommitOptions{
		Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
	})
	if err != nil && !errors.Is(err, git.ErrEmptyCommit) {
		return err
	}
	return nil
}

func (GitProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	path := userDir(user, cfg.GetLocalGitPath())
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	if _, err := git.PlainOpen(path); err == nil {
		// repo already exists
		return nil
	} else if !errors.Is(err, git.ErrRepositoryNotExists) {
		return err
	}
	r, err := git.PlainInit(path, false)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	added := false
	if _, err := os.Stat(filepath.Join(path, "readme.md")); os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(path, "readme.md"), []byte("# bookmarks"), 0600); err != nil {
			return err
		}
		if _, err := wt.Add("readme.md"); err != nil {
			return err
		}
		added = true
	}
	if _, err := os.Stat(filepath.Join(path, ".gitignore")); os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(path, ".gitignore"), []byte(".password\n"), 0600); err != nil {
			return err
		}
		if _, err := wt.Add(".gitignore"); err != nil {
			return err
		}
		added = true
	}
	if added {
		_, err = wt.Commit("init", &git.CommitOptions{
			Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
		})
		if err != nil && !errors.Is(err, git.ErrEmptyCommit) {
			return err
		}
	}
	return nil
}

func (GitProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	path := userDir(user, cfg.GetLocalGitPath())
	if _, err := git.PlainOpen(path); err == nil {
		return true, nil
	} else if errors.Is(err, git.ErrRepositoryNotExists) {
		return false, nil
	} else {
		return false, err
	}
}

func passwordPath(user string, path string) string {
	h := sha256.Sum256([]byte(user))
	return filepath.Join(path, hex.EncodeToString(h[:]), ".password")
}

// CreateUser writes a bcrypt hash for the given user. It returns ErrUserExists
// if the password file already exists.
func (GitProvider) CreateUser(ctx context.Context, user, password string) error {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	gp := GitProvider{}
	if err := gp.CreateRepo(ctx, user, nil, cfg.GetRepoName()); err != nil {
		return err
	}
	p := passwordPath(user, cfg.GetLocalGitPath())
	if _, err := os.Stat(p); err == nil {
		return ErrUserExists
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return os.WriteFile(p, hash, 0600)
}

// SetPassword updates the password for an existing user. ErrUserNotFound is
// returned when no password file exists.
func (GitProvider) SetPassword(ctx context.Context, user, password string) error {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	p := passwordPath(user, cfg.GetLocalGitPath())
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return ErrUserNotFound
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return os.WriteFile(p, hash, 0600)
}

// CheckPassword verifies the provided password against the stored bcrypt hash.
func (GitProvider) CheckPassword(ctx context.Context, user, password string) (bool, error) {
	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	data, err := os.ReadFile(passwordPath(user, cfg.GetLocalGitPath()))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if bcrypt.CompareHashAndPassword(data, []byte(password)) != nil {
		return false, nil
	}
	return true, nil
}
