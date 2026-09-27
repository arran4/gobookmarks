package main

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
)

func setupTestGlobalStateRestoration(t *testing.T) {
	t.Helper()

	originalConfig := gobookmarks.Config
	originalSessionStore := gobookmarks.SessionStore

	originalOrder := append([]string(nil), gobookmarks.ProviderNames()...)
	originalConfigOrder := append([]string(nil), gobookmarks.Config.ProviderOrder...)

	originalProviders := make(map[string]gobookmarks.Provider)
	for _, name := range originalOrder {
		originalProviders[name] = gobookmarks.GetProvider(name)
	}

	t.Cleanup(func() {
		// First clear out anything
		for _, name := range gobookmarks.ProviderNames() {
			gobookmarks.UnregisterProvider(name)
		}

		// Then restore the original ones
		for _, name := range originalOrder {
			if p := originalProviders[name]; p != nil {
				gobookmarks.RegisterProvider(p)
			}
		}

		gobookmarks.SetProviderOrder(originalOrder)
		gobookmarks.Config = originalConfig
		gobookmarks.Config.ProviderOrder = originalConfigOrder
		gobookmarks.SessionStore = originalSessionStore
	})
}

func TestScenarioDBFailureRestoration(t *testing.T) {
	// DO NOT USE t.Parallel()

	// 1. Set up unconditional restoration for all state
	setupTestGlobalStateRestoration(t)

	// Ensure SQL is unregistered to test full restoration in the face of modifications
	gobookmarks.UnregisterProvider("sql")

	testOrder := []string{"gitlab", "github", "git"}
	gobookmarks.SetProviderOrder(testOrder)
	gobookmarks.Config.ProviderOrder = []string{"some", "other", "order"} // Different from live

	if strings.Join(gobookmarks.ProviderNames(), ",") != "gitlab,github,git" {
		t.Fatalf("Failed to setup test live provider order: %v", gobookmarks.ProviderNames())
	}

	// Capture the original providers to compare against during restoration check
	originalProviders := make(map[string]gobookmarks.Provider)
	for _, name := range gobookmarks.ProviderNames() {
		originalProviders[name] = gobookmarks.GetProvider(name)
	}
	originalSessionStore := gobookmarks.SessionStore
	originalConfig := gobookmarks.Config

	// 2. Inject failure mechanism directly returning an error on OpenDB
	injectedOpenDBOverride := func() (*sql.DB, error) {
		return nil, errors.New("injected OpenDB failure")
	}

	// 3. Call setupScenarioBackend and expect it to fail
	cleanup, err := setupScenarioBackendWithOpenDB(injectedOpenDBOverride)

	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatalf("Expected setupScenarioBackend to fail due to injected error")
	}

	// 4. Verify exact restoration of the state THAT WAS PRESENT IMMEDIATELY PRIOR TO CALLING setupScenarioBackend()
	finalOrder := gobookmarks.ProviderNames()
	if strings.Join(finalOrder, ",") != "gitlab,github,git" {
		t.Errorf("ProviderNames not restored correctly. Expected 'gitlab,github,git', got '%v'", finalOrder)
	}

	for _, name := range finalOrder {
		if gobookmarks.GetProvider(name) != originalProviders[name] {
			t.Errorf("Provider %s was replaced/mutated, identity mismatch", name)
		}
	}

	if strings.Join(gobookmarks.Config.ProviderOrder, ",") != "some,other,order" {
		t.Errorf("Config.ProviderOrder changed. Expected 'some,other,order', got '%v'", gobookmarks.Config.ProviderOrder)
	}
	if gobookmarks.Config.LocalGitPath != originalConfig.LocalGitPath ||
		gobookmarks.Config.DBConnectionProvider != originalConfig.DBConnectionProvider ||
		gobookmarks.Config.DBConnectionString != originalConfig.DBConnectionString ||
		gobookmarks.Config.SessionKey != originalConfig.SessionKey ||
		gobookmarks.Config.SessionName != originalConfig.SessionName {
		t.Errorf("Config was not fully restored. Current: %+v", gobookmarks.Config)
	}

	if gobookmarks.SessionStore != originalSessionStore {
		t.Errorf("SessionStore not restored")
	}

	if gobookmarks.GetProvider("sql") != nil {
		t.Errorf("SQL provider should have been unregistered on failure")
	}
}

func TestScenarioDBTeardownRestoration(t *testing.T) {
	// DO NOT USE t.Parallel()

	// 1. Set up unconditional restoration for all state
	setupTestGlobalStateRestoration(t)

	// Ensure SQL is unregistered to test full restoration
	gobookmarks.UnregisterProvider("sql")

	testOrder := []string{"gitlab", "github", "git"}
	gobookmarks.SetProviderOrder(testOrder)
	gobookmarks.Config.ProviderOrder = []string{"some", "other", "order"} // Different from live

	if strings.Join(gobookmarks.ProviderNames(), ",") != "gitlab,github,git" {
		t.Fatalf("Failed to setup test live provider order: %v", gobookmarks.ProviderNames())
	}

	// Capture the original providers to compare against during restoration check
	originalProviders := make(map[string]gobookmarks.Provider)
	for _, name := range gobookmarks.ProviderNames() {
		originalProviders[name] = gobookmarks.GetProvider(name)
	}
	originalSessionStore := gobookmarks.SessionStore
	originalConfig := gobookmarks.Config

	// 3. Call setupScenarioBackend and expect it to succeed
	cleanup, err := setupScenarioBackend()

	if err != nil {
		t.Fatalf("Expected setupScenarioBackend to succeed: %v", err)
	}

	if cleanup == nil {
		t.Fatalf("Expected cleanup function to be returned")
	}

	cleanup() // TEARDOWN

	// 4. Verify exact restoration of the state THAT WAS PRESENT IMMEDIATELY PRIOR TO CALLING setupScenarioBackend()
	finalOrder := gobookmarks.ProviderNames()
	if strings.Join(finalOrder, ",") != "gitlab,github,git" {
		t.Errorf("ProviderNames not restored correctly. Expected 'gitlab,github,git', got '%v'", finalOrder)
	}

	for _, name := range finalOrder {
		if gobookmarks.GetProvider(name) != originalProviders[name] {
			t.Errorf("Provider %s was replaced/mutated, identity mismatch", name)
		}
	}

	if strings.Join(gobookmarks.Config.ProviderOrder, ",") != "some,other,order" {
		t.Errorf("Config.ProviderOrder changed. Expected 'some,other,order', got '%v'", gobookmarks.Config.ProviderOrder)
	}

	if gobookmarks.Config.LocalGitPath != originalConfig.LocalGitPath ||
		gobookmarks.Config.DBConnectionProvider != originalConfig.DBConnectionProvider ||
		gobookmarks.Config.DBConnectionString != originalConfig.DBConnectionString ||
		gobookmarks.Config.SessionKey != originalConfig.SessionKey ||
		gobookmarks.Config.SessionName != originalConfig.SessionName {
		t.Errorf("Config was not fully restored. Current: %+v", gobookmarks.Config)
	}

	if gobookmarks.SessionStore != originalSessionStore {
		t.Errorf("SessionStore not restored")
	}

	if gobookmarks.GetProvider("sql") != nil {
		t.Errorf("SQL provider should have been unregistered on failure")
	}
}
