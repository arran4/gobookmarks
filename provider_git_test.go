package gobookmarks

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestGitProviderCreateAndGet(t *testing.T) {
	tmp := t.TempDir()
	LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	text := "Category: Test\nhttp://example.com test"
	if err := p.CreateRepo(context.Background(), user, nil, RepoName); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", text); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	got, sha, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	if got != text {
		t.Fatalf("expected %q got %q", text, got)
	}
	if sha == "" {
		t.Fatalf("sha empty")
	}
}

func TestGitPasswordLifecycle(t *testing.T) {
	tmp := t.TempDir()
	LocalGitPath = tmp
	p := GitProvider{}
	ctx := context.Background()
	user := "bob"
	pass := "secret"

	// create user
	if err := p.CreateUser(ctx, user, pass); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	// creating again should fail
	if err := p.CreateUser(ctx, user, pass); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}

	ok, err := p.CheckPassword(ctx, user, pass)
	if err != nil || !ok {
		t.Fatalf("check password failed: %v %v", ok, err)
	}
	ok, err = p.CheckPassword(ctx, user, "wrong")
	if err != nil || ok {
		t.Fatalf("wrong password check: %v %v", ok, err)
	}

	// change password
	newPass := "newpass"
	if err := p.SetPassword(ctx, user, newPass); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	ok, _ = p.CheckPassword(ctx, user, newPass)
	if !ok {
		t.Fatalf("new password not accepted")
	}

	// setting password for missing user should fail
	if err := p.SetPassword(ctx, "missing", "pwd"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGitUserDirHash(t *testing.T) {
	LocalGitPath = "/base"
	path := userDir("../../etc/passwd")
	if filepath.Dir(path) != LocalGitPath {
		t.Fatalf("path escaped base: %s", path)
	}
	if filepath.Base(path) == "../../etc/passwd" {
		t.Fatalf("user dir not hashed")
	}
}
