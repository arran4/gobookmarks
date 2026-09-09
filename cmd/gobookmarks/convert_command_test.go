package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
)

func TestConvertCommand(t *testing.T) {
	validText := `Tab: Dashboard
Page: Main
Category: Search Engines
https://www.google.com Google Search
https://duckduckgo.com DuckDuckGo
Column
Category: Version Control
https://github.com GitHub
--
Category: Other
http://example.com`

	dir := t.TempDir()
	txtFile := filepath.Join(dir, "bookmarks.txt")
	jsonFile := filepath.Join(dir, "bookmarks.json")

	if err := os.WriteFile(txtFile, []byte(validText), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Text to JSON conversion
	cmd := NewRootCommand()

	// capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", txtFile})
	if err != nil {
		t.Fatalf("conversion from txt to json failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	jsonOutput := buf.Bytes()

	if err := os.WriteFile(jsonFile, jsonOutput, 0644); err != nil {
		t.Fatal(err)
	}

	// Verify valid JSON
	var parsedTabs []*gobookmarks.JSONTab
	if err := json.Unmarshal(jsonOutput, &parsedTabs); err != nil {
		t.Fatalf("output is not valid json: %v", err)
	}
	if len(parsedTabs) != 1 || parsedTabs[0].Name != "Dashboard" {
		t.Errorf("expected tab 'Dashboard', got %+v", parsedTabs)
	}

	// 2. JSON to Text conversion (round trip)
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	err = cmd.Execute([]string{"convert", "--from", "json", "--to", "bookmarks", jsonFile})
	if err != nil {
		t.Fatalf("conversion from json to txt failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	io.Copy(&buf, r)
	txtOutput := buf.String()

	if !strings.Contains(txtOutput, "Tab: Dashboard") || !strings.Contains(txtOutput, "https://github.com GitHub") {
		t.Errorf("round trip text output missing expected content:\n%s", txtOutput)
	}

	// Verify the round trip text parses without error and gives same structure
	// since we enforce semantic equality.
	list, err := gobookmarks.StrictParseBookmarks(txtOutput)
	if err != nil {
		t.Fatalf("round-tripped text failed strict parse: %v", err)
	}
	if len(list) != 1 || len(list[0].Pages) != 1 || list[0].Name != "Dashboard" {
		t.Errorf("semantic mismatch in round trip")
	}
}
