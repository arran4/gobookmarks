package main

import (
	"crypto/rand"
	"fmt"
	"os"

	gobookmarks "github.com/arran4/gobookmarks"
)

func setupScenarioBackend() (func(), error) {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	uuid := fmt.Sprintf("%x", id)

	tmpGitDir, err := os.MkdirTemp("", "scenario-git-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp git dir: %w", err)
	}

	gobookmarks.Config.LocalGitPath = tmpGitDir
	gobookmarks.Config.DBConnectionProvider = "sqlite3"
	gobookmarks.Config.DBConnectionString = "file:scenario-" + uuid + "?mode=memory&cache=shared"

	// Ensure providers are explicitly set and initialized in order
	gobookmarks.SetProviderOrder([]string{"sql", "git", "github", "gitlab"})

	gobookmarks.Config.SessionKey = "temp-scenario-key-that-is-long-enough-32bytes"
	gobookmarks.Config.SessionName = "scenario_session"
	gobookmarks.SessionStore = gobookmarks.InitSessionStore([]byte(gobookmarks.Config.SessionKey))

	// Ensure SQLProvider exists and is registered. Serve logic registers default providers if empty,
	// but we must ensure they exist in testing without panics.
	sqlP := &gobookmarks.SQLProvider{}
	gobookmarks.RegisterProvider(sqlP)

	gitP := &gobookmarks.GitProvider{}
	gobookmarks.RegisterProvider(gitP)

	githubP := &gobookmarks.GitHubProvider{}
	gobookmarks.RegisterProvider(githubP)

	gitlabP := &gobookmarks.GitLabProvider{}
	gobookmarks.RegisterProvider(gitlabP)

	// OpenDB will ping and call ensureSQLSchema to create tables
	db, err := gobookmarks.OpenDB()
	if err != nil {
		if cleanupErr := os.RemoveAll(tmpGitDir); cleanupErr != nil {
			return nil, fmt.Errorf("failed to initialize temp scenario db: %w (also failed to remove %s: %v)", err, tmpGitDir, cleanupErr)
		}
		return nil, fmt.Errorf("failed to initialize temp scenario db: %w", err)
	}
	defer func() { _ = db.Close() }()

	cleanup := func() {
		if err := os.RemoveAll(tmpGitDir); err != nil {
			// The caller cannot act on cleanup failures, but they must remain visible.
			fmt.Fprintf(os.Stderr, "scenario cleanup %s: %v\n", tmpGitDir, err)
		}
	}

	return cleanup, nil
}
