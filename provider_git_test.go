package gobookmarks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitProviderCreateAndGet(t *testing.T) {
	tmp := t.TempDir()
	AppConfig.LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	text := "Category: Test\nhttp://example.com test"
	if err := p.CreateRepo(context.Background(), user, nil, AppConfig.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(userDir(user), ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore missing: %v", err)
	}
	if !strings.Contains(string(gi), ".password") {
		t.Fatalf(".password not ignored")
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

func TestGitRepoExists(t *testing.T) {
	tmp := t.TempDir()
	AppConfig.LocalGitPath = tmp
	p := GitProvider{}
	user := "carol"
	exists, err := p.RepoExists(context.Background(), user, nil, AppConfig.GetRepoName())
	if err != nil {
		t.Fatalf("RepoExists before create: %v", err)
	}
	if exists {
		t.Fatalf("repo should not exist")
	}
	if err := p.CreateRepo(context.Background(), user, nil, AppConfig.GetRepoName()); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	exists, err = p.RepoExists(context.Background(), user, nil, AppConfig.GetRepoName())
	if err != nil || !exists {
		t.Fatalf("repo should exist, got %v %v", exists, err)
	}
}

func TestGitPasswordLifecycle(t *testing.T) {
	tmp := t.TempDir()
	AppConfig.LocalGitPath = tmp
	p := GitProvider{}
	ctx := context.Background()
	user := "bob"
	pass := "secret"

	if err := p.CreateUser(ctx, user, pass); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := os.Stat(filepath.Join(userDir(user), ".git")); err != nil {
		t.Fatalf("repo missing after CreateUser: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(userDir(user), ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore missing: %v", err)
	}
	if !strings.Contains(string(gi), ".password") {
		t.Fatalf(".password not ignored")
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
	AppConfig.LocalGitPath = "/base"
	path := userDir("../../etc/passwd")
	if filepath.Dir(path) != AppConfig.LocalGitPath {
		t.Fatalf("path escaped base: %s", path)
	}
	if filepath.Base(path) == "../../etc/passwd" {
		t.Fatalf("user dir not hashed")
	}
}
