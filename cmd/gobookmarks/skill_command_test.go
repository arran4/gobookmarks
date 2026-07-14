package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillCommandIntegration(t *testing.T) {
	// Setup test environment
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	// Create a local skill to install
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Integration Skill"), 0644)

	// Initialize root command
	root := NewRootCommand()

	// Test Installation
	err := root.Execute([]string{"skill", "install", "--scope", "user", "--agent", "common", srcDir, "int-skill"})
	if err != nil {
		t.Fatalf("skill install failed: %v", err)
	}

	// Verify installation happened
	destDir := filepath.Join(configHome, "agents", "skills", "int-skill")
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Fatalf("Skill directory not found at expected path: %s", destDir)
	}

	// Test Doctor
	err = root.Execute([]string{"skill", "doctor", "--scope", "user", "--agent", "common"})
	if err != nil {
		t.Fatalf("skill doctor failed: %v", err)
	}

	// Capture stdout output isn't trivial with this structure without redirecting os.Stdout,
	// but we can at least ensure the commands don't error out.

	// Test List
	err = root.Execute([]string{"skill", "list", "--scope", "user", "--agent", "common"})
	if err != nil {
		t.Fatalf("skill list failed: %v", err)
	}

	// Test Inspect
	err = root.Execute([]string{"skill", "inspect", "--scope", "user", "--agent", "common", "int-skill"})
	if err != nil {
		t.Fatalf("skill inspect failed: %v", err)
	}

	// Test Update (local update shouldn't fail even if forced)
	err = root.Execute([]string{"skill", "update", "--scope", "user", "--agent", "common", "--force", "int-skill"})
	if err != nil {
		t.Fatalf("skill update failed: %v", err)
	}

	// Test Remove
	err = root.Execute([]string{"skill", "remove", "--scope", "user", "--agent", "common", "int-skill"})
	if err != nil {
		t.Fatalf("skill remove failed: %v", err)
	}

	// Verify removal
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Fatalf("Skill directory still exists after remove: %s", destDir)
	}
}

func TestSkillInstallCommandInvalidSource(t *testing.T) {
	root := NewRootCommand()
	err := root.Execute([]string{"skill", "install", "/non/existent/path/for/sure/invalid", "bad-skill"})
	if err == nil {
		t.Fatalf("Expected error for invalid source, got nil")
	}
	if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "stat") {
		t.Fatalf("Expected not found error, got: %v", err)
	}
}
