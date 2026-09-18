package main

import (
	"strings"
	"testing"
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

func TestTempSQLiteBackend(t *testing.T) {
	err := setupTempSQLiteBackend()
	if err != nil {
		t.Fatalf("setupTempSQLiteBackend failed: %v", err)
	}
}
