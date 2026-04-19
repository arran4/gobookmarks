package gobookmarks

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"path"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/txtar/*.txtar
var testdataFS embed.FS

type TestParams struct {
	Index   int    `json:"index,omitempty"`
	NewText string `json:"newText,omitempty"`
	TabIdx  int    `json:"tabIdx,omitempty"`
	PageIdx int    `json:"pageIdx,omitempty"`
	TabName string `json:"tabName,omitempty"`
	OldName string `json:"oldName,omitempty"`
	NewName string `json:"newName,omitempty"`
}

func TestTxtarCases(t *testing.T) {
	entries, err := fs.Glob(testdataFS, "testdata/txtar/*.txtar")
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}

	for _, fixture := range entries {
		fixture := fixture
		t.Run(strings.TrimSuffix(path.Base(fixture), ".txtar"), func(t *testing.T) {
			raw, err := testdataFS.ReadFile(fixture)
			if err != nil {
				t.Fatalf("read fixture %s: %v", fixture, err)
			}
			ar := txtar.Parse(raw)

			var input, expected, testName string
			var params TestParams
			testName = "ParseBookmarks" // default

			for _, f := range ar.Files {
				if f.Name == "input.txt" {
					input = string(bytes.TrimSpace(f.Data))
				}
				if f.Name == "expected.txt" {
					expected = string(bytes.TrimSpace(f.Data))
				}
				if f.Name == "test.txt" {
					testName = string(bytes.TrimSpace(f.Data))
				}
				if f.Name == "params.json" {
					if err := json.Unmarshal(f.Data, &params); err != nil {
						t.Fatalf("failed to unmarshal params.json: %v", err)
					}
				}
			}

			var got string

			switch testName {
			case "ParseBookmarks":
				list := ParseBookmarks(input)
				got = strings.TrimSpace(list.String())
			case "ExtractCategoryByIndex":
				res, err := ExtractCategoryByIndex(input, params.Index)
				if err != nil {
					t.Fatalf("ExtractCategoryByIndex error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "ReplaceCategoryByIndex":
				res, err := ReplaceCategoryByIndex(input, params.Index, params.NewText)
				if err != nil {
					t.Fatalf("ReplaceCategoryByIndex error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "ExtractPage":
				res, name, err := ExtractPage(input, params.TabIdx, params.PageIdx)
				if err != nil {
					t.Fatalf("ExtractPage error: %v", err)
				}
				got = strings.TrimSpace(res) + "\nNAME=" + name
			case "ExtractTab":
				res, err := ExtractTab(input, params.TabName)
				if err != nil {
					t.Fatalf("ExtractTab error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "ExtractTabByIndex":
				res, err := ExtractTabByIndex(input, params.Index)
				if err != nil {
					t.Fatalf("ExtractTabByIndex error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "ReplaceTab":
				res, err := ReplaceTab(input, params.OldName, params.NewName, params.NewText)
				if err != nil {
					t.Fatalf("ReplaceTab error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "ReplaceTabByIndex":
				res, err := ReplaceTabByIndex(input, params.Index, params.NewName, params.NewText)
				if err != nil {
					t.Fatalf("ReplaceTabByIndex error: %v", err)
				}
				got = strings.TrimSpace(res)
			case "AppendTab":
				res := AppendTab(input, params.NewName, params.NewText)
				got = strings.TrimSpace(res)
			default:
				t.Fatalf("unknown test name: %s", testName)
			}

			if got != expected {
				t.Errorf("\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s\n", input, expected, got)
			}
		})
	}
}
