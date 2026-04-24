package gobookmarks

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBookmarksJSParser(t *testing.T) {
	// Read the JS script we created
	jsFile, err := os.ReadFile("templates/bookmarks_parser.gohtml")
	if err != nil {
		t.Fatalf("Failed to read JS template file: %v", err)
	}

	// Extract just the javascript part (remove gohtml tags)
	jsCode := string(jsFile)
	jsCode = strings.ReplaceAll(jsCode, `{{define "bookmarks_parser"}}`, "")
	jsCode = strings.ReplaceAll(jsCode, `{{end}}`, "")
	jsCode = strings.ReplaceAll(jsCode, `<script>`, "")
	jsCode = strings.ReplaceAll(jsCode, `</script>`, "")

	// We'll write a node script that loads these functions and tests them against our sample text
	testScript := jsCode + `
const sampleBookmarks = ` + "`" + `Title: My Bookmarks

Tab: Tab 1
Page: Page 1
Category: Cat 1
- Link 1
- Link 2

Category: Cat 2
- Link 3

Page: Page 2
Category: Cat 3
- Link 4

Tab: Tab 2
Page: Page 3
Category: Cat 4
- Link 5
--
` + "`" + `;

function assertEqual(actual, expected, message) {
    if (actual.trim() !== expected.trim()) {
        console.error("FAILED: " + message);
        console.error("Expected:\n" + expected);
        console.error("Actual:\n" + actual);
        process.exit(1);
    }
}

// Test extractTabByIndex
let tab1 = extractTabByIndex(sampleBookmarks, 0);
assertEqual(tab1, "Tab: Tab 1\nPage: Page 1\nCategory: Cat 1\n- Link 1\n- Link 2\n\nCategory: Cat 2\n- Link 3\n\nPage: Page 2\nCategory: Cat 3\n- Link 4\n\n", "Extract Tab 1");

let tab2 = extractTabByIndex(sampleBookmarks, 1);
assertEqual(tab2, "Tab: Tab 2\nPage: Page 3\nCategory: Cat 4\n- Link 5\n--\n", "Extract Tab 2");

// Test extractPage
let page1 = extractPage(sampleBookmarks, 0, 0);
assertEqual(page1, "Page: Page 1\nCategory: Cat 1\n- Link 1\n- Link 2\n\nCategory: Cat 2\n- Link 3", "Extract Tab 1, Page 1");

// Test extractCategoryByIndex
let cat1 = extractCategoryByIndex(sampleBookmarks, 0, 0, 0);
assertEqual(cat1, "Category: Cat 1\n- Link 1\n- Link 2", "Extract Tab 1, Page 1, Cat 1");

let cat2 = extractCategoryByIndex(sampleBookmarks, 0, 0, 1);
assertEqual(cat2, "Category: Cat 2\n- Link 3", "Extract Tab 1, Page 1, Cat 2");

console.log("All JS parser tests passed!");
`

	err = os.WriteFile("test_js_parser.js", []byte(testScript), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary test script: %v", err)
	}
	defer func() { _ = os.Remove("test_js_parser.js") }()

	cmd := exec.Command("node", "test_js_parser.js")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("JS parser test failed:\nError: %v\nStdout: %s\nStderr: %s", err, out.String(), stderr.String())
	}

	if !strings.Contains(out.String(), "All JS parser tests passed!") {
		t.Fatalf("Unexpected output from JS parser test: %s", out.String())
	}
}
