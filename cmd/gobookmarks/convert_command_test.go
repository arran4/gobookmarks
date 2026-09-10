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

	// This tests explicit unnamed tab vs implicit unnamed tab
	explicitUnnamedTabText := `Tab
Page: Main
Category: Search Engines
https://www.google.com Google Search`

	dir := t.TempDir()
	txtFile := filepath.Join(dir, "bookmarks.txt")
	explicitUnnamedTxtFile := filepath.Join(dir, "explicit_unnamed.txt")
	jsonFile := filepath.Join(dir, "bookmarks.json")
	explicitUnnamedJsonFile := filepath.Join(dir, "explicit_unnamed.json")
	invalidJsonFile := filepath.Join(dir, "invalid.json")

	if err := os.WriteFile(invalidJsonFile, []byte(`[{"pages": [{"blocks": [null]}]}]`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(txtFile, []byte(validText), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(explicitUnnamedTxtFile, []byte(explicitUnnamedTabText), 0644); err != nil {
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

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
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

	_ = w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	_, _ = io.Copy(&buf, r)
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

	// 3. Test explicit unnamed tab round trip
	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", explicitUnnamedTxtFile})
	if err != nil {
		t.Fatalf("conversion of explicit unnamed txt to json failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	var explicitUnnamedJsonBuf bytes.Buffer
	_, _ = io.Copy(&explicitUnnamedJsonBuf, r)
	if err := os.WriteFile(explicitUnnamedJsonFile, explicitUnnamedJsonBuf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd.Execute([]string{"convert", "--from", "json", "--to", "bookmarks", explicitUnnamedJsonFile})
	if err != nil {
		t.Fatalf("conversion of explicit unnamed json to txt failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	explicitUnnamedTxtOutputBuf := new(bytes.Buffer)
	_, _ = io.Copy(explicitUnnamedTxtOutputBuf, r)
	explicitUnnamedTxtOutput := explicitUnnamedTxtOutputBuf.String()

	if !strings.HasPrefix(strings.TrimSpace(explicitUnnamedTxtOutput), "Tab") {
		t.Fatalf("round-tripped text lost explicit unnamed Tab directive:\n%s", explicitUnnamedTxtOutput)
	}

	// 4. Test implicit unnamed tab
	implicitUnnamedText := `Page: Main
Category: Search Engines
https://www.google.com Google Search`
	implicitUnnamedTxtFile := filepath.Join(dir, "implicit_unnamed.txt")
	if err := os.WriteFile(implicitUnnamedTxtFile, []byte(implicitUnnamedText), 0644); err != nil {
		t.Fatal(err)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", implicitUnnamedTxtFile})
	if err != nil {
		t.Fatalf("conversion of implicit unnamed txt to json failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	var implicitUnnamedJsonBuf bytes.Buffer
	_, _ = io.Copy(&implicitUnnamedJsonBuf, r)
	implicitUnnamedJsonFile := filepath.Join(dir, "implicit_unnamed.json")
	if err := os.WriteFile(implicitUnnamedJsonFile, implicitUnnamedJsonBuf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd.Execute([]string{"convert", "--from", "json", "--to", "bookmarks", implicitUnnamedJsonFile})
	if err != nil {
		t.Fatalf("conversion of implicit unnamed json to txt failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	implicitUnnamedTxtOutputBuf := new(bytes.Buffer)
	_, _ = io.Copy(implicitUnnamedTxtOutputBuf, r)
	implicitUnnamedTxtOutput := implicitUnnamedTxtOutputBuf.String()

	if strings.Contains(implicitUnnamedTxtOutput, "Tab") {
		t.Fatalf("round-tripped text incorrectly gained explicit Tab directive:\n%s", implicitUnnamedTxtOutput)
	}

	// 5. Test Invalid Semantic JSON
	err = cmd.Execute([]string{"convert", "--from", "json", "--to", "bookmarks", invalidJsonFile})
	if err == nil {
		t.Fatal("expected invalid semantic json to fail conversion")
	}

	// 6. Test config-independence under a deliberately unusable setup
	os.Setenv("DB_CONNECTION_PROVIDER", "invalid_provider_that_would_crash_if_loaded")
	os.Setenv("LOCAL_GIT_PATH", "/dev/null/invalid_git_path")
	defer os.Unsetenv("DB_CONNECTION_PROVIDER")
	defer os.Unsetenv("LOCAL_GIT_PATH")

	// Create a new RootCommand to pick up the env changes just in case, though convert should skip loadConfig
	cmdNoConfig := NewRootCommand()
	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmdNoConfig.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", txtFile})
	if err != nil {
		t.Fatalf("conversion should succeed without loading application config: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	// Test lint config independence
	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmdNoConfig.Execute([]string{"lint", txtFile})
	if err != nil {
		t.Fatalf("lint should succeed without loading application config: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	// 7. Test Unnamed pages vs named pages round trip (in explicitUnnamedTxtOutput test)
	// explicitUnnamedText has "Page: Main" which is named.
	// let's test unnamed page
	unnamedPageText := `Tab: Dashboard
Page
Category: Search
https://google.com`
	unnamedPageTxtFile := filepath.Join(dir, "unnamed_page.txt")
	if err := os.WriteFile(unnamedPageTxtFile, []byte(unnamedPageText), 0644); err != nil {
		t.Fatal(err)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd.Execute([]string{"convert", "--from", "bookmarks", "--to", "bookmarks", unnamedPageTxtFile})
	if err != nil {
		t.Fatalf("conversion of unnamed page txt to bookmarks failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	unnamedPageTxtOutputBuf := new(bytes.Buffer)
	_, _ = io.Copy(unnamedPageTxtOutputBuf, r)
	unnamedPageTxtOutput := unnamedPageTxtOutputBuf.String()

	// It should retain "Page"
	if !strings.Contains(unnamedPageTxtOutput, "Page\n") {
		t.Fatalf("round-tripped text lost unnamed Page directive:\n%s", unnamedPageTxtOutput)
	}
}
