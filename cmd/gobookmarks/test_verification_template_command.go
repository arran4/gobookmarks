package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	. "github.com/arran4/gobookmarks"
)

type TemplateCommand struct {
	parent Command
	Flags  *flag.FlagSet

	DataFromJsonFile stringFlag
	Serve            stringFlag
	Out              stringFlag
	HelpCmd          *HelpCommand
}

func (c *VerificationCommand) NewTemplateCommand() (*TemplateCommand, error) {
	tc := &TemplateCommand{
		parent: c,
		Flags:  flag.NewFlagSet("template", flag.ContinueOnError),
	}
	tc.Flags.Var(&tc.DataFromJsonFile, "data-from-json-file", "Path to JSON file containing template data")
	tc.Flags.Var(&tc.Serve, "serve", "Address to serve the template on (e.g. :8080)")
	tc.Flags.Var(&tc.Out, "out", "File to write output to")
	tc.HelpCmd = NewHelpCommand(tc)
	return tc, nil
}

func (c *TemplateCommand) Name() string {
	return c.Flags.Name()
}

func (c *TemplateCommand) Parent() Command {
	return c.parent
}

func (c *TemplateCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *TemplateCommand) Subcommands() []Command {
	return []Command{c.HelpCmd}
}

func (c *TemplateCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		return c.HelpCmd.Execute(nil)
	}

	subcommandRange := remaining[0]
	// Handle help within template command if user types "gobookmarks test verification template help"
	if subcommandRange == "help" || subcommandRange == "-h" || subcommandRange == "--help" {
		return c.HelpCmd.Execute(remaining[1:])
	}

	coreData := &CoreData{
		Title:    "Test Verification",
		UserRef:  "testuser",
		EditMode: false,
		Tab:      0,
	}

	var bookmarksStr string
	templateName := "mainPage.gohtml"
	switch subcommandRange {
	case "default":
		bookmarksStr = `
Tab: Default Tab
Page: Default Page
Category: Default Category
https://example.com Example Link
`
	case "complex":
		bookmarksStr = `
Tab: Tab 1
Page: Page 1
Category: Search Engines
https://google.com Google
https://bing.com Bing
Column
Category: Social Media
https://twitter.com Twitter
https://facebook.com Facebook
--
Category: News
https://news.ycombinator.com Hacker News
https://reddit.com Reddit

Tab: Tab 2
Page: Page 2
Category: Coding
https://github.com GitHub
https://stackoverflow.com Stack Overflow
`
	case "edit":
		templateName = "edit.gohtml"
		bookmarksStr = `
Tab: Default Tab
Page: Default Page
Category: Default Category
https://example.com Example Link
`
		coreData.EditMode = true
	default:
		// If unknown range, maybe treat it as empty or minimal
		bookmarksStr = "Tab: Empty\n"
	}

	if c.DataFromJsonFile.set {
		// Load data from JSON file
		data, err := os.ReadFile(c.DataFromJsonFile.value)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}

		type JsonInput struct {
			Title     string `json:"title"`
			UserRef   string `json:"user_ref"`
			EditMode  bool   `json:"edit_mode"`
			Bookmarks string `json:"bookmarks"`
		}
		var input JsonInput
		if err := json.Unmarshal(data, &input); err != nil {
			return fmt.Errorf("failed to parse json data: %w", err)
		}
		if input.Title != "" {
			coreData.Title = input.Title
		}
		if input.UserRef != "" {
			coreData.UserRef = input.UserRef
		}
		coreData.EditMode = input.EditMode
		if input.Bookmarks != "" {
			bookmarksStr = input.Bookmarks
		}
	} else {
		// Just to debug if set is true or not
		// fmt.Println("DEBUG: DataFromJsonFile is NOT set")
	}

	// Create a dummy request to build the context
	req, _ := http.NewRequest("GET", "/", nil)
	// We need to pass coreData so that middleware-like access works,
	// but NewFuncs uses r.Context().Value(ContextValues("coreData")) potentially?
	// Actually NewFuncs doesn't seem to use coreData directly, but templates do via {{ $.Title }}
	// The templates receive `data` which has `CoreData`.

	// However, some funcs might need session or other context values.
	// For "useCssColumns" etc.

	ctx := context.WithValue(req.Context(), ContextValues("coreData"), coreData)
	req = req.WithContext(ctx)

	// Create funcs that override the default behavior to return our static data
	funcs := NewFuncs(req)

	// Override specific functions to use our local bookmarks string
	funcs["bookmarks"] = func() (string, error) {
		return bookmarksStr, nil
	}
	funcs["bookmarksExist"] = func() (bool, error) {
		return bookmarksStr != "", nil
	}
	funcs["bookmarkPages"] = func() ([]*BookmarkPage, error) {
		tabs := ParseBookmarks(bookmarksStr)
		idx := TabFromRequest(req)
		if idx < 0 || idx >= len(tabs) {
			idx = 0
		}
		if len(tabs) == 0 {
			return nil, nil
		}
		return tabs[idx].Pages, nil
	}
	funcs["bookmarkTabs"] = func() ([]TabInfo, error) {
		tabsData := ParseBookmarks(bookmarksStr)
		var tabs []TabInfo
		for i, t := range tabsData {
			indexName := t.DisplayName()
			if indexName == "" && i == 0 {
				indexName = "Main"
			}
			if indexName != "" {
				href := TabHref(i, "") // No ref in static mode
				lastSha := "" // No SHA in static mode
				if len(t.Pages) > 0 {
					lastSha = t.Pages[len(t.Pages)-1].Sha()
				}
				tabs = append(tabs, TabInfo{
					Index: i,
					Name: t.Name,
					IndexName: indexName,
					Href: href,
					EditHref: AppendQueryParams(href, "edit", "1"),
					LastPageSha: lastSha,
				})
			}
		}
		return tabs, nil
	}
	funcs["bookmarkTabsWithPages"] = func() ([]TabWithPages, error) {
		tabsData := ParseBookmarks(bookmarksStr)
		var tabs []TabWithPages
		for i, t := range tabsData {
			indexName := t.DisplayName()
			if indexName == "" && i == 0 {
				indexName = "Main"
			}
			if indexName != "" {
				href := TabHref(i, "")
				lastSha := ""
				if len(t.Pages) > 0 {
					lastSha = t.Pages[len(t.Pages)-1].Sha()
				}
				tabs = append(tabs, TabWithPages{
					TabInfo: TabInfo{
						Index: i,
						Name: t.Name,
						IndexName: indexName,
						Href: href,
						EditHref: AppendQueryParams(href, "edit", "1"),
						LastPageSha: lastSha,
					},
					Pages:   t.Pages,
				})
			}
		}
		return tabs, nil
	}
	funcs["tabName"] = func() string {
		tabs := ParseBookmarks(bookmarksStr)
		idx := TabFromRequest(req)
		if idx < 0 || idx >= len(tabs) {
			idx = 0
		}
		if len(tabs) == 0 {
			return ""
		}
		name := tabs[idx].DisplayName()
		if name == "" && idx == 0 {
			name = "Main"
		}
		return name
	}
	// Override other funcs that might call DB/Git
	funcs["loggedIn"] = func() (bool, error) { return true, nil }
	funcs["showPages"] = func() bool { return true }


	// Override additional functions for edit pages
	funcs["bookmarksOrEditBookmarks"] = func() (string, error) {
		return bookmarksStr, nil
	}
	funcs["branchOrEditBranch"] = func() string { return "main" }
	funcs["ref"] = func() string { return "sha123" }
	funcs["bookmarksSHA"] = func() string { return "sha123" }

	// Compile templates with our modified funcs
	tmpl := GetCompiledTemplates(funcs)

	type Data struct {
		*CoreData
		Error string
	}
	data := Data{
		CoreData: coreData,
		Error:    "",
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	output := buf.Bytes()

	if c.Out.set {
		if err := os.WriteFile(c.Out.value, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.Printf("Output written to %s", c.Out.value)
	}

	if c.Serve.set {
		log.Printf("Serving template on %s", c.Serve.value)

		// For serving, we need to handle main.css and favicon too, otherwise the page looks broken
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(output)
		})
		mux.HandleFunc("/main.css", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			w.Write(GetMainCSSData())
		})
		mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Write(GetFavicon())
		})
		// Also proxy/favicon if possible, but that might require internet or network
		mux.HandleFunc("/proxy/favicon", func(w http.ResponseWriter, r *http.Request) {
			// Mock or minimal implementation
			w.WriteHeader(http.StatusNotFound)
		})

		return http.ListenAndServe(c.Serve.value, mux)
	}

	// If neither out nor serve is set, write to stdout?
	if !c.Out.set && !c.Serve.set {
		fmt.Println(string(output))
	}

	return nil
}
