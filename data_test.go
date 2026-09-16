package gobookmarks

import (
	"bytes"
	"html/template"
	"io"
	"net/http/httptest"
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
		"asset":              func(p string) (string, error) { return AssetURL(p) },
		"assetURL":           func(p string) (string, error) { return AssetURL(p) },
		"version":            func() string { return "test" },
		"CurrentURL":         func() string { return "/" },
		"LoginPageURL":       func() string { return "https://example.com/login" },
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
		"tabEditPath":    func(tab int) string { return TabEditPath(tab) },
		"tabEditHref":    func(tab int, ref, name string) string { return TabEditHref(tab, ref, name) },
		"currentTabPath": func() string { return "/" },
		"appendQuery":    func(rawURL string, params ...string) string { return AppendQueryParams(rawURL, params...) },
		"tabName":        func() string { return "Main" },
		"page":           func() string { return "" },
		"historyRef":     func() string { return "refs/heads/main" },
		"devMode":        func() bool { return false },
		"showFooter":     func() bool { return true },
		"showPages":      func() bool { return true },
		"loggedIn":       func() (bool, error) { return true, nil },
		"bookmarkTabs": func() ([]TabInfo, error) {
			return []TabInfo{{Index: 0, Name: "", IndexName: "Main", Href: "/", LastPageSha: ""}}, nil
		},
		"bookmarkTabsWithPages": func() ([]TabWithPages, error) {
			pages := []*BookmarkPage{
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
			}
			return []TabWithPages{{
				TabInfo: TabInfo{Index: 0, Name: "", IndexName: "Main", Href: "/", LastPageSha: ""},
				Pages:   pages,
			}}, nil
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
		"taskSaveAndDone": func() string {
			return TaskSaveAndDone
		},
		"taskSaveAndStopEditing": func() string { return TaskSaveAndStopEditing },
		"jsMode":                 func() string { return "" },
		"appJs":                  func() template.JS { return "" },
		"useCssColumns":          func() bool { return false },
	}
}

func TestExecuteTemplates(t *testing.T) {
	tpl := template.New("").Funcs(testFuncMap())
	tpl, err := ParseFSRecursive(tpl, os.DirFS("./templates"), ".", ".gohtml")
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






func TestAppJSRenderModes(t *testing.T) {
	// Only parse tail.gohtml to avoid full func map requirements
	tpl := template.New("tail")
	b, _ := os.ReadFile("templates/tail.gohtml")

	// Add required dummy functions used specifically in tail.gohtml
	funcs := testFuncMap()
	funcs["asset"] = func(s string) (string, error) { return "/assets/" + s, nil }
	funcs["jsMode"] = func() string { return "" }
	funcs["appJs"] = func() template.JS { return "" }
	funcs["branchOrEditBranch"] = func() string { return "" }
	funcs["ref"] = func() string { return "" }
	funcs["bookmarksSHA"] = func() string { return "" }
	funcs["taskSaveAndStopEditing"] = func() string { return "" }

	tpl, err := tpl.Funcs(funcs).Parse(string(b))
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	appJsData := GetAppJSData()

	runTest := func(name, jsQuery string, expectedInline bool) {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?js="+jsQuery, nil)

			clone, _ := tpl.Clone()

			// Override specific funcs for this run
			runFuncs := template.FuncMap{
				"jsMode": func() string { return req.URL.Query().Get("js") },
				"appJs": func() template.JS { return template.JS(appJsData) },
			}
			clone = clone.Funcs(runFuncs)

			var buf bytes.Buffer
			err := clone.Execute(&buf, map[string]any{
				"loggedIn": true,
			})
			if err != nil {
				t.Fatalf("execute error: %v", err)
			}

			out := buf.String()

			if expectedInline {
				if !strings.Contains(out, "<script type=\"module\">") {
					t.Errorf("expected inline script tag")
				}
				if !strings.Contains(out, string(appJsData)) {
					t.Errorf("expected inline source code")
				}
				if strings.Contains(out, "src=\"/assets/web/app") {
					t.Errorf("did not expect asset src link")
				}
			} else {
				if !strings.Contains(out, "import { initApp }") {
					t.Errorf("expected module import block")
				}
				if !strings.Contains(out, "import { initApp } from ") || !strings.Contains(out, "assets") || !strings.Contains(out, "app.mjs") {
					t.Errorf("expected fingerprinted URL usage\nGot:\n%s", out)
				}
				if strings.Contains(out, string(appJsData)) {
					t.Errorf("did not expect full inline source code")
				}
			}
		})
	}

	runTest("default inline", "", true)
	runTest("explicit inline", "inline", true)
	runTest("asset mode", "asset", false)
	runTest("unknown fallback to inline", "unknown", true)
}
