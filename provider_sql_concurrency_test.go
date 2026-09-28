package gobookmarks

import (
	"context"
	"strings"
	"testing"
)

func TestSQLProvider_UpdateBookmarks_Concurrency(t *testing.T) {
	oldProv := Config.DBConnectionProvider
	oldStr := Config.DBConnectionString
	defer func() {
		Config.DBConnectionProvider = oldProv
		Config.DBConnectionString = oldStr
	}()

	Config.DBConnectionProvider = "sqlite3"
	Config.DBConnectionString = ":memory:"

	p := &SQLProvider{}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	user := "testuser"
	branch := "main"
	initialText := "initial text"

	err := p.CreateRepo(ctx, user, nil, "repo")
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}

	err = p.CreateBookmarks(ctx, user, nil, branch, initialText)
	if err != nil {
		t.Fatalf("CreateBookmarks failed: %v", err)
	}

	_, sha, err := p.GetBookmarks(ctx, user, branch, nil)
	if err != nil {
		t.Fatalf("GetBookmarks failed: %v", err)
	}

	// Update with a bad sha should fail
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text", "badsha")
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") {
		t.Fatalf("Expected sha mismatch error, got: %v", err)
	}

	// Update with a good sha should succeed
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 2", sha)
	if err != nil {
		t.Fatalf("Expected update to succeed with good sha, got: %v", err)
	}

	// Another update with the OLD sha (stale update) should fail
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 3", sha)
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") {
		t.Fatalf("Expected sha mismatch error on stale sha, got: %v", err)
	}

	// Empty sha should succeed (empty-expectSHA compatibility)
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 4", "")
	if err != nil {
		t.Fatalf("Expected empty-expectSHA to succeed, got: %v", err)
	}

	// Empty expectSHA on absent branch should succeed
	err = p.UpdateBookmarks(ctx, user, nil, "newbranch", "newbranch", "new branch text", "")
	if err != nil {
		t.Fatalf("Expected empty-expectSHA on absent branch to succeed, got: %v", err)
	}

	// Non-empty expectSHA on absent branch should fail as conflict
	err = p.UpdateBookmarks(ctx, user, nil, "absentbranch", "absentbranch", "new text", "somesha")
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") {
		t.Fatalf("Expected sha mismatch error on absent branch with non-empty expectSHA, got: %v", err)
	}
}

func TestSQLProvider_UpdateBookmarks_Rollback(t *testing.T) {
	oldProv := Config.DBConnectionProvider
	oldStr := Config.DBConnectionString
	defer func() {
		Config.DBConnectionProvider = oldProv
		Config.DBConnectionString = oldStr
	}()

	Config.DBConnectionProvider = "sqlite3"
	Config.DBConnectionString = ":memory:"

	p := &SQLProvider{}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	user := "testuser"
	branch := "main"
	initialText := "initial text"

	err := p.CreateRepo(ctx, user, nil, "repo")
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}

	err = p.CreateBookmarks(ctx, user, nil, branch, initialText)
	if err != nil {
		t.Fatalf("CreateBookmarks failed: %v", err)
	}

	_, sha1, err := p.GetBookmarks(ctx, user, branch, nil)
	if err != nil {
		t.Fatalf("GetBookmarks failed: %v", err)
	}

	// Update with good sha should succeed
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 2", sha1)
	if err != nil {
		t.Fatalf("Expected update to succeed with good sha, got: %v", err)
	}

	text2, sha2, _ := p.GetBookmarks(ctx, user, branch, nil)

	// Another update with the OLD sha (stale update) should fail
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 3", sha1)
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") {
		t.Fatalf("Expected sha mismatch error on stale sha, got: %v", err)
	}

	text3, sha3, _ := p.GetBookmarks(ctx, user, branch, nil)

	if text3 != text2 {
		t.Fatalf("Expected text to be unchanged after failed update, got: %s", text3)
	}

	if sha3 != sha2 {
		t.Fatalf("Expected sha to be unchanged after failed update, got: %s", sha3)
	}

	// Ensure history was not written
	commits, err := p.GetCommits(ctx, user, nil, branch, 1, 100)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	if len(commits) != 2 {
		// 1 from create, 1 from the successful update
		t.Fatalf("Expected 2 commits, got %d", len(commits))
	}
}
