package main

import (
	"context"
	"os"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
)

func TestParseScenario(t *testing.T) {
	txtar := `Test preamble
-- scenario.meta --
Name: test

-- 01-user.event --
Op: user.create
Username: testuser
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	if s.Preamble != "Test preamble\n" {
		t.Errorf("expected preamble 'Test preamble\n', got %q", s.Preamble)
	}

	if s.Manifest["Name"] != "test" {
		t.Errorf("expected manifest Name 'test', got %q", s.Manifest["Name"])
	}

	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.Events))
	}

	if s.Events[0].Op != "user.create" {
		t.Errorf("expected event Op 'user.create', got %q", s.Events[0].Op)
	}
}

func TestScenarioValidation(t *testing.T) {
	txtar := `
-- 01-user.event --
Op: user.create
` // Missing Username

	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	err = ValidateScenario(s)
	if err == nil {
		t.Errorf("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "missing Username") {
		t.Errorf("expected missing Username error, got: %v", err)
	}
}

func TestBookmarkAssetResolution(t *testing.T) {
	txtar := `
-- 010-bookmarks.event --
Op: bookmark.create
User: user
Asset: missing.txt
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}
	err = ValidateScenario(s) // Validation passes, failure happens during Apply due to dynamic Context
	if err != nil {
		t.Fatalf("expected validation success: %v", err)
	}

	sCtx := &ScenarioContext{
		Refs:  map[string]string{"user": "resolvedUser"},
		Files: s.Files,
	}

	op := operations["bookmark.create"]
	err = op.Apply(context.Background(), sCtx, s.Events[0])
	if err == nil {
		t.Errorf("expected missing asset error")
	}
	if !strings.Contains(err.Error(), "missing asset: missing.txt") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnknownOperation(t *testing.T) {
	txtar := `
-- 010-unknown.event --
Op: unknown.op
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}
	err = ValidateScenario(s)
	if err == nil {
		t.Errorf("expected error for unknown op")
	}
}

func TestSymbolicReferenceFailure(t *testing.T) {
	txtar := `
-- 010-repo.event --
Op: repo.create
User: missing-ref
Name: main
`
	s, err := ParseScenario(strings.NewReader(txtar))
	if err != nil {
		t.Fatalf("ParseScenario failed: %v", err)
	}

	sCtx := &ScenarioContext{
		Refs:  map[string]string{},
		Files: s.Files,
	}

	op := operations["repo.create"]
	err = op.Apply(context.Background(), sCtx, s.Events[0])
	if err == nil {
		t.Errorf("expected unknown user ref error")
	}
	if !strings.Contains(err.Error(), "unknown user ref") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTempSQLiteBackend(t *testing.T) {
	err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}
}

func TestHistoryScenario(t *testing.T) {
	err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}

	scenarioPath := "scenarios/history.txtar"
	file, err := os.Open(scenarioPath)
	if err != nil {
		t.Fatalf("Failed to open scenario %s: %v", scenarioPath, err)
	}
	defer func() { _ = file.Close() }()

	scenario, err := ParseScenario(file)
	if err != nil {
		t.Fatalf("Failed to parse scenario: %v", err)
	}

	if err := ValidateScenario(scenario); err != nil {
		t.Fatalf("Failed to validate scenario: %v", err)
	}

	if err := ApplyScenario(context.Background(), scenario); err != nil {
		t.Fatalf("Failed to apply scenario: %v", err)
	}

	// Verify history via provider
	p := gobookmarks.GetProvider("sql")
	commits, err := p.GetCommits(context.Background(), "bob", nil, "main", 1, 10)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	// We expect 3 commits from the history scenario
	if len(commits) != 3 {
		t.Errorf("Expected 3 commits, got %d", len(commits))
	}
}
