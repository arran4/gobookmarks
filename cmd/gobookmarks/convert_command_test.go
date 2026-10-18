package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gobookmarks "github.com/arran4/gobookmarks"
)

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}

func TestConvertLogic(t *testing.T) {
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

	explicitUnnamedTabText := `Tab
Page: Main
Category: Search Engines
https://www.google.com Google Search`

	implicitUnnamedText := `Page: Main
Category: Search Engines
https://www.google.com Google Search`

	unnamedPageText := `Tab: Dashboard
Page
Category: Search
https://google.com`

	t.Run("Text to JSON to Text", func(t *testing.T) {
		var buf bytes.Buffer
		err := Convert("bookmarks", "json", strings.NewReader(validText), &buf)
		if err != nil {
			t.Fatalf("text to json failed: %v", err)
		}

		jsonOutput := buf.Bytes()
		var parsedTabs []*gobookmarks.JSONTab
		if err := json.Unmarshal(jsonOutput, &parsedTabs); err != nil {
			t.Fatalf("output is not valid json: %v", err)
		}
		if len(parsedTabs) != 1 || parsedTabs[0].Name != "Dashboard" {
			t.Errorf("expected tab 'Dashboard', got %+v", parsedTabs)
		}

		var buf2 bytes.Buffer
		err = Convert("json", "bookmarks", bytes.NewReader(jsonOutput), &buf2)
		if err != nil {
			t.Fatalf("json to text failed: %v", err)
		}

		txtOutput := buf2.String()
		if !strings.Contains(txtOutput, "Tab: Dashboard") || !strings.Contains(txtOutput, "https://github.com GitHub") {
			t.Errorf("round trip text output missing expected content:\n%s", txtOutput)
		}
	})

	t.Run("Explicit Unnamed Tab", func(t *testing.T) {
		var buf bytes.Buffer
		err := Convert("bookmarks", "json", strings.NewReader(explicitUnnamedTabText), &buf)
		if err != nil {
			t.Fatalf("text to json failed: %v", err)
		}

		var buf2 bytes.Buffer
		err = Convert("json", "bookmarks", &buf, &buf2)
		if err != nil {
			t.Fatalf("json to text failed: %v", err)
		}

		if !strings.HasPrefix(strings.TrimSpace(buf2.String()), "Tab") {
			t.Fatalf("round-tripped text lost explicit unnamed Tab directive:\n%s", buf2.String())
		}
	})

	t.Run("Implicit Unnamed Tab", func(t *testing.T) {
		var buf bytes.Buffer
		err := Convert("bookmarks", "json", strings.NewReader(implicitUnnamedText), &buf)
		if err != nil {
			t.Fatalf("text to json failed: %v", err)
		}

		var buf2 bytes.Buffer
		err = Convert("json", "bookmarks", &buf, &buf2)
		if err != nil {
			t.Fatalf("json to text failed: %v", err)
		}

		if strings.Contains(buf2.String(), "Tab") {
			t.Fatalf("round-tripped text incorrectly gained explicit Tab directive:\n%s", buf2.String())
		}
	})

	t.Run("Invalid Semantic JSON", func(t *testing.T) {
		invalidJson := `[{"pages": [{"blocks": [null]}]}]`
		var buf bytes.Buffer
		err := Convert("json", "bookmarks", strings.NewReader(invalidJson), &buf)
		if err == nil {
			t.Fatal("expected invalid semantic json to fail conversion")
		}
	})

	t.Run("Unnamed Pages vs Named Pages Round Trip", func(t *testing.T) {
		var buf bytes.Buffer
		err := Convert("bookmarks", "json", strings.NewReader(unnamedPageText), &buf)
		if err != nil {
			t.Fatalf("text to json failed: %v", err)
		}

		var buf2 bytes.Buffer
		err = Convert("json", "bookmarks", &buf, &buf2)
		if err != nil {
			t.Fatalf("json to text failed: %v", err)
		}

		txtOutput := buf2.String()
		list, err := gobookmarks.StrictParseBookmarks(txtOutput)
		if err != nil {
			t.Fatalf("round-tripped unnamed page text failed strict parse: %v", err)
		}
		if len(list) != 1 || len(list[0].Pages) != 2 {
			t.Fatalf("semantic mismatch in unnamed page round trip length: got %d pages", len(list[0].Pages))
		}
		if list[0].Pages[0].Name != "" || list[0].Pages[1].Name != "" {
			t.Fatalf("unnamed page became named: %q", list[0].Pages[1].Name)
		}
	})

	invalidLossyJsonFiles := []struct {
		Name string
		JSON string
	}{
		{"hr block with columns", `[{"pages": [{"blocks": [{"hr": true, "columns": [{}]}]}]}]`},
		{"empty category name", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": ""}]}]}]}]}]`},
		{"empty entry url", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": "Test", "entries": [{"url": ""}]}]}]}]}]}]`},
		{"named but implicit tab", `[{"name": "Named", "explicit": false}]`},
		{"non-first implicit tab", `[{}, {"explicit": false}]`},
		{"empty blocks array", `[{"pages": [{"blocks": []}]}]`},
		{"empty pages array", `[{"pages": []}]`},
		{"degenerate implicit empty model", `[{}]`},
		{"leading HR block", `[{"explicit":true,"pages":[{"blocks":[{"hr":true},{"columns":[{}]}]}]}]`},
		{"trailing HR block", `[{"explicit":true,"pages":[{"blocks":[{"columns":[{}]},{"hr":true}]}]}]`},
		{"consecutive HR blocks", `[{"explicit":true,"pages":[{"blocks":[{"columns":[{}]},{"hr":true},{"hr":true},{"columns":[{}]}]}]}]`},
		{"adjacent non-HR blocks", `[{"explicit":true,"pages":[{"blocks":[{"columns":[{}]},{"columns":[{}]}]}]}]`},
		{"empty display name", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": "C", "entries": [{"url": "https://example.com", "name": ""}]}]}]}]}]}]`},
		{"whitespace inside entry url", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": "C", "entries": [{"url": "foo bar", "name": "N"}]}]}]}]}]}]`},
		{"empty implicit first tab dropping", `[{}, {"explicit":true,"name":"Second Tab"}]`},
		{"empty unnamed first page dropping", `[{"pages": [{}, {"name":"Named Second Page"}]}]`},
		{"multiple top-level values", `[{"explicit":true}] [{"explicit":true}]`},
		{"trailing non-whitespace garbage", `[{"explicit":true}] xyz`},
		{"leading space in category name", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": " C"}]}]}]}]}]`},
		{"trailing space in entry url", `[{"pages": [{"blocks": [{"columns": [{"categories": [{"name": "C", "entries": [{"url": "https://example.com "}]}]}]}]}]}]`},
	}

	for _, tc := range invalidLossyJsonFiles {
		t.Run("Lossy Invalid JSON: "+tc.Name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Convert("json", "bookmarks", strings.NewReader(tc.JSON), &buf)
			if err == nil {
				t.Fatalf("expected lossy semantic json (%s) to fail conversion", tc.Name)
			}
		})
	}

	t.Run("Failing Reader Boundary", func(t *testing.T) {
		var buf bytes.Buffer
		err := Convert("bookmarks", "json", errReader{}, &buf)
		if err == nil || !strings.Contains(err.Error(), "simulated read error") {
			t.Fatalf("expected simulated read error, got: %v", err)
		}
	})

	t.Run("Failing Writer Boundary", func(t *testing.T) {
		err := Convert("bookmarks", "json", strings.NewReader(validText), errWriter{})
		if err == nil || !strings.Contains(err.Error(), "simulated write error") {
			t.Fatalf("expected simulated write error, got: %v", err)
		}
	})
}

func TestConvertCommand_CLI(t *testing.T) {
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

	altText := `Tab: NotDashboard
Page: Main
Category: Search
https://google.com Google`

	dir := t.TempDir()
	txtFile := filepath.Join(dir, "bookmarks.txt")
	mustWriteFile(t, txtFile, []byte(validText))

	t.Run("Named file argument", func(t *testing.T) {
		cmdNoConfig := NewRootCommand()
		cmdNoConfig.ConvertCmd.Stdin = strings.NewReader(altText) // Stdin should be ignored in favor of the named file
		var buf bytes.Buffer
		cmdNoConfig.ConvertCmd.Stdout = &buf

		err := cmdNoConfig.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", txtFile})
		if err != nil {
			t.Fatalf("conversion should succeed without loading application config: %v", err)
		}

		var parsedTabs []*gobookmarks.JSONTab
		if err := json.Unmarshal(buf.Bytes(), &parsedTabs); err != nil {
			t.Fatalf("output is not valid json: %v", err)
		}
		if len(parsedTabs) != 1 || parsedTabs[0].Name != "Dashboard" {
			t.Fatalf("expected named file data ('Dashboard'), got: %s", buf.String())
		}
	})

	t.Run("Explicit stdin -", func(t *testing.T) {
		cmdNoConfig := NewRootCommand()
		cmdNoConfig.ConvertCmd.Stdin = strings.NewReader(altText)
		var buf bytes.Buffer
		cmdNoConfig.ConvertCmd.Stdout = &buf

		err := cmdNoConfig.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", "-"})
		if err != nil {
			t.Fatalf("conversion failed with explicit stdin dash: %v", err)
		}

		var parsedTabs []*gobookmarks.JSONTab
		if err := json.Unmarshal(buf.Bytes(), &parsedTabs); err != nil {
			t.Fatalf("output is not valid json: %v", err)
		}
		if len(parsedTabs) != 1 || parsedTabs[0].Name != "NotDashboard" {
			t.Fatalf("expected explicit stdin data ('NotDashboard'), got: %s", buf.String())
		}
	})

	t.Run("Implicit stdin via zero args", func(t *testing.T) {
		cmdNoConfig := NewRootCommand()
		cmdNoConfig.ConvertCmd.Stdin = strings.NewReader(altText)
		var buf bytes.Buffer
		cmdNoConfig.ConvertCmd.Stdout = &buf

		err := cmdNoConfig.Execute([]string{"convert", "--from", "bookmarks", "--to", "json"})
		if err != nil {
			t.Fatalf("conversion failed with implicit stdin: %v", err)
		}

		var parsedTabs []*gobookmarks.JSONTab
		if err := json.Unmarshal(buf.Bytes(), &parsedTabs); err != nil {
			t.Fatalf("output is not valid json: %v", err)
		}
		if len(parsedTabs) != 1 || parsedTabs[0].Name != "NotDashboard" {
			t.Fatalf("expected implicit stdin data ('NotDashboard'), got: %s", buf.String())
		}
	})

	t.Run("Lint config independence without breaking stdout", func(t *testing.T) {
		// Verify convert config independence too
		t.Setenv("GOBM_CONFIG_FILE", "/dev/null/invalid_config_that_does_not_exist_xyz.json")

		cmdNoConfig := NewRootCommand()
		var buf bytes.Buffer
		cmdNoConfig.ConvertCmd.Stdout = &buf
		err := cmdNoConfig.Execute([]string{"convert", "--from", "bookmarks", "--to", "json", txtFile})
		if err != nil {
			t.Fatalf("conversion should succeed without loading application config: %v", err)
		}

		cmdNoConfig2 := NewRootCommand()
		// We do not mutate os.Stdout. We just verify the command completes
		err = cmdNoConfig2.Execute([]string{"lint", txtFile})
		if err != nil {
			t.Fatalf("lint should succeed without loading application config: %v", err)
		}
	})
}
