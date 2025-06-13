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
	"golang.org/x/oauth2"
)

// GitProvider implements Provider using local git repositories.
type GitProvider struct{}

func init() { RegisterProvider(GitProvider{}) }

func (GitProvider) Name() string                                                     { return "git" }
func (GitProvider) DefaultServer() string                                            { return "" }
func (GitProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config { return nil }
func (GitProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return &User{Login: "local"}, nil
}

func userDir(user string) string {
	h := sha256.Sum256([]byte(user))
	return filepath.Join(LocalGitPath, hex.EncodeToString(h[:]))
}

func openRepo(user string) (*git.Repository, error) {
	r, err := git.PlainOpen(userDir(user))
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return r, nil
}

func (GitProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	r, err := openRepo(user)
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
	r, err := openRepo(user)
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

func (GitProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	r, err := openRepo(user)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, nil
		}
		return nil, err
	}
	ref, err := r.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		return nil, nil
	}
	iter, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}
	var commits []*Commit
	err = iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, &Commit{
			SHA:            c.Hash.String(),
			Message:        c.Message,
			CommitterName:  c.Committer.Name,
			CommitterEmail: c.Committer.Email,
			CommitterDate:  c.Committer.When,
		})
		return nil
	})
	return commits, err
}

func (GitProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	r, err := openRepo(user)
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
	r, err := openRepo(user)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch)})
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
	if err := os.WriteFile(filepath.Join(userDir(user), "bookmarks.txt"), []byte(text), 0600); err != nil {
		return err
	}
	if _, err := wt.Add("bookmarks.txt"); err != nil {
		return err
	}
	_, err = wt.Commit("Auto change from web", &git.CommitOptions{
		Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
	})
	return err
}

func (GitProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	r, err := openRepo(user)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Create: true})
	if err != nil {
		if !errors.Is(err, git.ErrBranchExists) {
			// branch already exists
			if err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch)}); err != nil {
				return err
			}
		}
	}
	if err := os.WriteFile(filepath.Join(userDir(user), "bookmarks.txt"), []byte(text), 0600); err != nil {
		return err
	}
	if _, err := wt.Add("bookmarks.txt"); err != nil {
		return err
	}
	_, err = wt.Commit("Auto create from web", &git.CommitOptions{
		Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
	})
	return err
}

func (GitProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	path := userDir(user)
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	r, err := git.PlainInit(path, false)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			r, err = git.PlainOpen(path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "readme.md"), []byte("# bookmarks"), 0600); err != nil {
		return err
	}
	if _, err := wt.Add("readme.md"); err != nil {
		return err
	}
	_, err = wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "Gobookmarks", Email: "Gobookmarks@arran.net.au", When: time.Now()},
	})
	return err
}

func passwordPath(user string) string {
	h := sha256.Sum256([]byte(user))
	return filepath.Join(LocalGitPath, hex.EncodeToString(h[:]), ".password")
}

// CreateUser writes a bcrypt hash for the given user. It returns ErrUserExists
// if the password file already exists.
func (GitProvider) CreateUser(ctx context.Context, user, password string) error {
	p := passwordPath(user)
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
	p := passwordPath(user)
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
	data, err := os.ReadFile(passwordPath(user))
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
