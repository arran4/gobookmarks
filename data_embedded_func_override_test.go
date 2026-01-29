//go:build !live
// +build !live

package gobookmarks

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arran4/gobookmarks/core"
)

func TestGetCompiledTemplates_FuncOverride(t *testing.T) {
	// Define a custom FuncMap with a specific override
	// We use the existing mock-like func map from data_test.go which is already
	// populated with safe dummy functions for testing.
	safeFuncs := testFuncMap()

	// And the specific override we want to test
	safeFuncs["version"] = func() string {
		return "OVERRIDDEN_VERSION"
	}

	// Get the compiled templates with the override applied
	tmpl := GetCompiledTemplates(safeFuncs)

	// Execute a template that uses {{ version }}
	// "loginPage.gohtml" typically uses {{ version }} in the footer
	var buf bytes.Buffer
	data := struct {
		*core.CoreData
		Error string
	}{
		CoreData: &core.CoreData{Title: "Test", UserRef: "user"},
	}

	if err := tmpl.ExecuteTemplate(&buf, "loginPage.gohtml", data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "OVERRIDDEN_VERSION") {
		t.Errorf("expected output to contain 'OVERRIDDEN_VERSION', but got:\n%s", output)
	}
}
