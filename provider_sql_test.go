package gobookmarks

import (
	"testing"
)

func TestSQLProviderClose(t *testing.T) {
	// Initialize a temporary SQL provider
	p := &SQLProvider{}

	// Create an in-memory db
	oldProv := Config.DBConnectionProvider
	oldStr := Config.DBConnectionString
	defer func() {
		Config.DBConnectionProvider = oldProv
		Config.DBConnectionString = oldStr
	}()

	Config.DBConnectionProvider = "sqlite3"
	Config.DBConnectionString = ":memory:"

	// Trigger DB initialization
	_, err := p.getDB()
	if err != nil {
		t.Fatalf("getDB failed: %v", err)
	}

	if p.db == nil {
		t.Fatalf("Expected SQLProvider.db to be initialized")
	}

	// Capture the raw DB connection pointer
	db := p.db

	// Ensure ping succeeds
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed before close: %v", err)
	}

	// Close the provider explicitly
	if err := p.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if p.db != nil {
		t.Errorf("Expected SQLProvider.db to be set to nil after Close")
	}

	// Verify that the underlying connection was actually closed
	if err := db.Ping(); err == nil {
		t.Errorf("Expected ping to fail on closed database, but it succeeded")
	}
}
