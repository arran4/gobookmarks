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

func TestSQLProvider_UpdateBookmarks_Rollback_LateFailure(t *testing.T) {
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

	// To test a failure AFTER branches has been updated but before the transaction commits,
	// we can cause the history insert to fail by passing a text that violates some constraint,
	// or by closing the DB in another thread, or by altering the schema dynamically.
	// Let's drop the bookmarks table right before we call UpdateBookmarks!
	// Wait, UpdateBookmarks does everything in one transaction, dropping the table would cause the final UPDATE bookmarks to fail.

	// Let's drop the `bookmarks` table!
	db, _ := p.getDB()
	_, err = db.Exec("DROP TABLE bookmarks")
	if err != nil {
		t.Fatalf("Drop table failed: %v", err)
	}

	// Now try to update. branches update will succeed, history will succeed, but bookmarks will fail.
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text 2", sha1)
	if err == nil {
		t.Fatalf("Expected update to fail due to missing bookmarks table")
	}

	// Now restore the table so we can check
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS bookmarks (user TEXT PRIMARY KEY, list BLOB)")
	if err != nil {
		t.Fatalf("Restore table failed: %v", err)
	}

	// Re-insert the original row
	_, err = db.Exec("INSERT INTO bookmarks (user, list) VALUES (?, ?)", user, initialText)
	if err != nil {
		t.Fatalf("Restore row failed: %v", err)
	}

	// Verify branches table was rolled back (sha should be sha1, not the new sha)
	var curSha string
	err = db.QueryRow("SELECT sha FROM branches WHERE user=? AND name=?", user, branch).Scan(&curSha)
	if err != nil {
		t.Fatalf("Query branches failed: %v", err)
	}

	if curSha != sha1 {
		t.Fatalf("Expected branches sha to be rolled back to %s, got %s", sha1, curSha)
	}

	// Ensure history was rolled back
	commits, err := p.GetCommits(ctx, user, nil, branch, 1, 100)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	if len(commits) != 1 {
		// Only 1 from create
		t.Fatalf("Expected 1 commit after rollback, got %d", len(commits))
	}
}

// The above rollback test covers late-stage failure and ensures history and branch sha are fully rolled back.

func TestSQLProvider_UpdateBookmarks_NullLegacy(t *testing.T) {
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

	// Artificially set the branch SHA to NULL to simulate legacy/unversioned states
	db, _ := p.getDB()
	_, err = db.Exec("UPDATE branches SET sha=NULL WHERE user=? AND name=?", user, branch)
	if err != nil {
		t.Fatalf("Set null sha failed: %v", err)
	}

	// Non-empty expectSHA on a NULL branch should fail as conflict
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text", "somesha")
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") {
		t.Fatalf("Expected sha mismatch error on null branch with non-empty expectSHA, got: %v", err)
	}

	// Empty expectSHA on a NULL branch should succeed (unversioned compatibility)
	err = p.UpdateBookmarks(ctx, user, nil, branch, branch, "new text", "")
	if err != nil {
		t.Fatalf("Expected empty-expectSHA on null branch to succeed, got: %v", err)
	}
}
