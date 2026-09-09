package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintCommand(t *testing.T) {
	rc := NewRootCommand()

	// valid file
	validText := `Tab: My Tab
Page: My Page
Category: Search
https://google.com Google`

	dir := t.TempDir()
	validFile := filepath.Join(dir, "valid.txt")
	if err := os.WriteFile(validFile, []byte(validText), 0644); err != nil {
		t.Fatal(err)
	}

	err := rc.Execute([]string{"lint", validFile})
	if err != nil {
		t.Fatalf("expected valid file to pass lint, got: %v", err)
	}

	// Invalid file (malformed directive/out of category)
	invalidText := `UnknownDirective
Tab: My Tab
Page: My Page
Category: Search
https://google.com Google
`
	invalidFile := filepath.Join(dir, "invalid.txt")
	if err := os.WriteFile(invalidFile, []byte(invalidText), 0644); err != nil {
		t.Fatal(err)
	}

	err = rc.Execute([]string{"lint", invalidFile})
	if err == nil {
		t.Fatal("expected invalid file to fail lint")
	}
	if !strings.Contains(err.Error(), "unrecognized directive") {
		t.Errorf("expected error about unrecognized directive, got: %v", err)
	}
}
