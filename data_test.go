package gobookmarks

import (
	"html/template"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCompileGoHTML(t *testing.T) {
	tpl, err := template.New("").Funcs(NewFuncs(nil)).ParseFS(os.DirFS("./templates"), "*.gohtml")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	files := []string{
		"edit.gohtml",
		"editCategory.gohtml",
		"editTab.gohtml",
		"editPage.gohtml",
		"editNotes.gohtml",
		"error.gohtml",
		"head.gohtml",
		"history.gohtml",
		"historyCommits.gohtml",
		"mainPage.gohtml",
		"loginPage.gohtml",
		"logoutPage.gohtml",
		"dragdrop.gohtml",
		"tail.gohtml",
		"taskDoneAutoRefreshPage.gohtml",
		"statusPage.gohtml",
	}

	for _, name := range files {
		if tpl.Lookup(name) == nil {
			t.Errorf("template %s not found", name)
		}
	}
}

func testFuncMap() template.FuncMap {
	return template.FuncMap{
		"now":                func() time.Time { return time.Unix(0, 0) },
		"version":            func() string { return "test" },
		"LoginURL":           func(p string) string { return "https://example.com/login/" + p },
		"Providers":          func() []string { return []string{"github", "gitlab"} },
		"AllProviders":       func() []string { return []string{"github", "gitlab"} },
		"ProviderConfigured": func(string) bool { return true },
		"errorMsg":           func(s string) string { return s },
		"ref":                func() string { return "refs/heads/main" },
		"add1":               func(i int) int { return i + 1 },
		"sub1": func(i int) int {
			if i > 0 {
				return i - 1
			}
			return 0
		},
		"atoi":           func(s string) int { i, _ := strconv.Atoi(s); return i },
		"tab":            func() string { return "0" },
                "tabPath":        func(tab int) string { return "/" },
                "tabEditPath":   func(tab int) string { return TabEditPath(tab) },
                "tabEditHref":   func(tab int, ref, name string) string { return TabEditHref(tab, ref, name) },
                "currentTabPath": func() string { return "/" },
		"appendQuery":    func(rawURL string, params ...string) string { return AppendQueryParams(rawURL, params...) },
		"tabName":        func() string { return "Main" },
		"page":           func() string { return "" },
		"historyRef":     func() string { return "refs/heads/main" },
		"useCssColumns":  func() bool { return false },
		"devMode":        func() bool { return false },
		"showFooter":     func() bool { return true },
		"showPages":      func() bool { return true },
		"loggedIn":       func() (bool, error) { return true, nil },
		"bookmarkTabs": func() ([]TabInfo, error) {
			return []TabInfo{{Index: 0, Name: "", IndexName: "Main", Href: "/", EditHref: "/?edit=1", LastPageSha: ""}}, nil
		},
		"commitShort": func() string {
			short := commit
			if len(short) > 7 {
				short = short[:7]
			}
			return short
		},
		"buildDate": func() string {
			return date
		},
		"bookmarkPages": func() ([]*BookmarkPage, error) {
			return []*BookmarkPage{
				{
					Blocks: []*BookmarkBlock{
						{
							Columns: []*BookmarkColumn{
								{
									Categories: []*BookmarkCategory{
										{
											Name:  "Demo",
											Index: 0,
											Entries: []*BookmarkEntry{
												{Name: "Home", Url: "https://example.com"},
											},
										},
									},
								},
							},
						},
					},
				},
			}, nil
		},
		"bookmarksOrEditBookmarks": func() (string, error) { return "Category: Demo\nhttps://example.com Home", nil },
		"bookmarksExist":           func() (bool, error) { return true, nil },
		"bookmarksSHA":             func() (string, error) { return "sha", nil },
		"branchOrEditBranch":       func() (string, error) { return "main", nil },
		"tags": func() ([]*Tag, error) {
			return []*Tag{{Name: "v1"}}, nil
		},
		"branches": func() ([]*Branch, error) {
			return []*Branch{{Name: "main"}}, nil
		},
		"commits": func() ([]*Commit, error) {
			return []*Commit{{
				SHA:            "abc",
				Message:        "msg",
				CommitterName:  "dev",
				CommitterEmail: "dev@example.com",
				CommitterDate:  time.Unix(0, 0),
			}}, nil
		},
		"prevCommit":  func() string { return "prev" },
		"nextCommit":  func() string { return "next" },
		"isSearchURL": func(string) bool { return false },
		"searchURL":   func(u string) string { return strings.TrimPrefix(u, "search:") },
		"taskSave":    func() string { return TaskSave },
		"taskSaveAndDone": func() string {
			return TaskSaveAndDone
		},
		"taskSaveAndStopEditing": func() string {
			return TaskSaveAndStopEditing
		},
	}
}

func TestExecuteTemplates(t *testing.T) {
	tpl, err := template.New("").Funcs(testFuncMap()).ParseFS(os.DirFS("./templates"), "*.gohtml")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}
	baseData := struct {
		*CoreData
		Error string
	}{
		CoreData: &CoreData{Title: "Test", UserRef: "user"},
	}

	catData := struct {
		*CoreData
		Error string
		Index int
		Text  string
		Sha   string
		Col   int
	}{
		CoreData: baseData.CoreData,
		Index:    0,
		Text:     "Category: Demo",
		Sha:      "sha",
		Col:      0,
	}

	pageData := struct {
		*CoreData
		Error string
		Name  string
		Text  string
		Sha   string
	}{
		CoreData: baseData.CoreData,
		Name:     "Demo",
		Text:     "Category: Demo",
		Sha:      "sha",
	}

	pages := []struct {
		name string
		tmpl string
		data any
	}{
		{"main", "mainPage.gohtml", baseData},
		{"login", "loginPage.gohtml", baseData},
		{"logout", "logoutPage.gohtml", baseData},
		{"edit", "edit.gohtml", baseData},
		{"editCategory", "editCategory.gohtml", catData},
		{"editPage", "editPage.gohtml", pageData},
		{"history", "history.gohtml", baseData},
		{"historyCommits", "historyCommits.gohtml", baseData},
		{"taskDone", "taskDoneAutoRefreshPage.gohtml", baseData},
		{"error", "error.gohtml", struct {
			*CoreData
			Error string
		}{baseData.CoreData, "boom"}},
	}

	for _, tt := range pages {
		t.Run(tt.name, func(t *testing.T) {
			if err := tpl.ExecuteTemplate(io.Discard, tt.tmpl, tt.data); err != nil {
				t.Errorf("execute %s: %v", tt.tmpl, err)
			}
		})
	}
}
