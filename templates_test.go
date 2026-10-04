package gobookmarks

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
)

func renderMainPageForColumnTest(t *testing.T, bookmarks string, cssColumns bool) string {
	t.Helper()

	list := ParseBookmarks(bookmarks)
	tabs := make([]TabWithPages, 0, len(list))
	for i, tab := range list {
		indexName := tab.DisplayName()
		if indexName == "" && i == 0 {
			indexName = "Main"
		}
		tabs = append(tabs, TabWithPages{
			TabInfo: TabInfo{
				Index:     i,
				Name:      tab.Name,
				IndexName: indexName,
				Href:      "/",
			},
			Pages: tab.Pages,
		})
	}

	funcs := testFuncMap()
	funcs["bookmarkTabsWithPages"] = func() ([]TabWithPages, error) { return tabs, nil }
	funcs["useCssColumns"] = func() bool { return cssColumns }
	funcs["bookmarksExist"] = func() (bool, error) { return true, nil }
	funcs["ref"] = func() string { return "refs/heads/main" }
	funcs["tab"] = func() string { return "0" }

	tmpl := template.New("").Funcs(funcs)
	tmpl, err := ParseFSRecursive(tmpl, os.DirFS("./templates"), ".", ".gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(
		&buf,
		"mainPage.gohtml",
		map[string]interface{}{
			"CoreData": &CoreData{Title: "Test", UserRef: "user"},
			"loggedIn": true,
		},
	); err != nil {
		t.Fatalf("execute mainPage.gohtml: %v", err)
	}
	return buf.String()
}

func TestMainPage_RenderingColumns(t *testing.T) {
	bookmarksStr := "Tab: A\nPage: B\nColumn: Alpha\nCategory: AlphaCat\nColumn\nCategory: BetaCat\nColumn: <script>&Name</script>\nCategory: GammaCat\nColumn: EmptyNamed\nColumn: Alpha\n"

	out := renderMainPageForColumnTest(t, bookmarksStr, false)

	if !strings.Contains(out, "<h2>Alpha <a class=\"edit-link\"") {
		t.Errorf("Expected normal named column heading to render.")
	}
	if !strings.Contains(out, "<h2>&lt;script&gt;&amp;Name&lt;/script&gt;") {
		t.Errorf("Expected escaped named column heading.")
	}
	if strings.Contains(out, "<h2><script>&Name</script>") {
		t.Fatal("column name rendered without HTML escaping")
	}
	if !strings.Contains(out, `<h2 class="unnamed-column-heading" style="display: none;"></h2>`) {
		t.Errorf("Expected unnamed heading container to be hidden.")
	}
	if !strings.Contains(out, `class="edit-mode-only"`) {
		t.Errorf("Expected edit-mode-only rename affordance for unnamed column.")
	}
	if !strings.Contains(out, `data-named="true"`) {
		t.Errorf("Expected data-named=\"true\" for named columns including empty ones.")
	}
}

func TestMainPage_RenderingCssColumns(t *testing.T) {
	bookmarksStr := "Tab: A\nPage: B\nColumn: Alpha\nCategory: AlphaCat\nColumn\nCategory: BetaCat\n"

	out := renderMainPageForColumnTest(t, bookmarksStr, true)

	if !strings.Contains(out, "cssColumns") {
		t.Errorf("Expected cssColumns class to be applied to page.")
	}
	if !strings.Contains(out, `<h2 class="unnamed-column-heading" style="display: none;"></h2>`) {
		t.Errorf("Expected unnamed heading container to be hidden in CSS columns view.")
	}
}
