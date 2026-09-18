package main

import (
	"crypto/rand"
	"fmt"

	gobookmarks "github.com/arran4/gobookmarks"
)

func setupScenarioBackend() error {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	uuid := fmt.Sprintf("%x", id)

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
		return fmt.Errorf("failed to initialize temp scenario db: %w", err)
	}
	defer func() { _ = db.Close() }()

	return nil
}
