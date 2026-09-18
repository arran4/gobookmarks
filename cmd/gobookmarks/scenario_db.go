package main

import (
	"crypto/rand"
	"fmt"

	gobookmarks "github.com/arran4/gobookmarks"
)

func setupTempSQLiteBackend() error {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	uuid := fmt.Sprintf("%x", id)

	gobookmarks.Config.DBConnectionProvider = "sqlite3"
	gobookmarks.Config.DBConnectionString = "file:scenario-" + uuid + "?mode=memory&cache=shared"
	gobookmarks.Config.ProviderOrder = []string{"sql"}

	// OpenDB will ping and call ensureSQLSchema to create tables
	db, err := gobookmarks.OpenDB()
	if err != nil {
		return fmt.Errorf("failed to initialize temp scenario db: %w", err)
	}
	defer func() { _ = db.Close() }()

	return nil
}
