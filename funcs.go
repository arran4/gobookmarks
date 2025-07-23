package gobookmarks

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TabInfo is used by templates to display tab navigation with indexes.
type TabInfo struct {
	Index       int
	Name        string
	IndexName   string
	Href        string
	LastPageSha string
}

var (
	defaultBookmarks = "Category: Example 1\nhttp://www.google.com.au Google\nColumn\nCategory: Example 2\nhttp://www.google.com.au Google\nhttp://www.google.com.au Google\n"
	version          = "dev"
	commit           = "none"
	date             = "unknown"
)

func SetVersion(pVersion, pCommit, pDate string) {
	version = pVersion
	commit = pCommit
	date = pDate
}

func NewFuncs(r *http.Request) template.FuncMap {
	return map[string]any{
		"now": func() time.Time { return time.Now() },
		"version": func() string {
			return version
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
		"firstline": func(s string) string {
			return strings.Split(s, "\n")[0]
		},
		"left": func(i int, s string) string {
			l := len(s)
			if l > i {
				l = i
			}
			return s[:l]
		},
		"LoginURL": func(p string) string {
			return "/login/" + p
		},
		"Providers": func() []string {
			names := make([]string, 0)
			for _, n := range ProviderNames() {
				creds := providerCreds(n)
				if creds != nil {
					names = append(names, n)
				}
			}
			return names
		},
		"AllProviders": func() []string {
			return ProviderNames()
		},
		"ProviderConfigured": func(p string) bool {
			creds := providerCreds(p)
			return creds != nil && GetProvider(p) != nil
		},
		"errorMsg": errorMessage,
		"ref": func() string {
			return r.URL.Query().Get("ref")
		},
		"tab": func() string {
			return r.URL.Query().Get("tab")
		},
		"page": func() string {
			return r.URL.Query().Get("page")
		},
		"historyRef": func() string {
			return r.URL.Query().Get("historyRef")
		},
		"add1": func(i int) int {
			return i + 1
		},
		"sub1": func(i int) int {
			if i > 0 {
				return i - 1
			}
			return 0
		},
		"atoi": func(s string) int {
			i, _ := strconv.Atoi(s)
			return i
		},
		"useCssColumns": func() bool {
			sessioni := r.Context().Value(ContextValues("session"))
			if session, ok := sessioni.(*sessions.Session); ok && session != nil {
				if v, ok := session.Values["useCssColumns"].(bool); ok {
					return v
				}
			}
			return UseCssColumns
		},
		"devMode": func() bool {
			return DevMode
		},
		"showFooter": func() bool {
			return !NoFooter
		},
		"showPages": func() bool {
			if r == nil {
				return false
			}
			if strings.HasPrefix(r.URL.Path, "/login") || r.URL.Path == "/status" {
				return false
			}
			sessioni := r.Context().Value(ContextValues("session"))
			session, ok := sessioni.(*sessions.Session)
			if !ok || session == nil {
				return false
			}
			githubUser, ok := session.Values["GithubUser"].(*User)
			if !ok || githubUser == nil {
				return false
			}
			return true
		},
		"loggedIn": func() (bool, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, ok := session.Values["GithubUser"].(*User)
			return ok && githubUser != nil, nil
		},
		"bookmarks": func() (string, error) {
			return Bookmarks(r)
		},
		"bookmarksOrEditBookmarks": func() (string, error) {
			if r.PostFormValue("text") != "" {
				return r.PostFormValue("text"), nil
			}
			return Bookmarks(r)
		},
		"bookmarksExist": func() (bool, error) {
			return BookmarksExist(r)
		},
		"bookmarksSHA": func() (string, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)
			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}
			_, sha, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					return "", nil
				}
				return "", err
			}
			return sha, nil
		},
		"branchOrEditBranch": func() (string, error) {
			if r.PostFormValue("branch") != "" {
				return r.PostFormValue("branch"), nil
			}
			ref := r.URL.Query().Get("ref")
			if strings.HasPrefix(ref, "refs/heads/") {
				return strings.TrimPrefix(ref, "refs/heads/"), nil
			} else if strings.HasPrefix(ref, "refs/tags/") {
				return "New" + strings.TrimPrefix(ref, "refs/tags/"), nil
			} else if len(ref) > 0 {
				return "FromCommit" + ref, nil
			}
			return "main", nil
		},
		"bookmarkPages": func() ([]*BookmarkPage, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}

			bookmarks, _, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			var bookmark = defaultBookmarks
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					bookmark = ""
				} else {
					return nil, fmt.Errorf("bookmarkPages: %w", err)
				}
			} else {
				bookmark = bookmarks
			}
			tabs := ParseBookmarks(bookmark)
			tabStr := r.URL.Query().Get("tab")
			idx, err := strconv.Atoi(tabStr)
			if err != nil || idx < 0 || idx >= len(tabs) {
				idx = 0
			}
			return tabs[idx].Pages, nil
		},
		"bookmarkTabs": func() ([]TabInfo, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}

			bookmarks, _, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			var bookmark = defaultBookmarks
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					bookmark = ""
				} else {
					return nil, fmt.Errorf("bookmarkTabs: %w", err)
				}
			} else {
				bookmark = bookmarks
			}
			tabsData := ParseBookmarks(bookmark)
			var tabs []TabInfo
			for i, t := range tabsData {
				indexName := t.DisplayName()
				if indexName == "" && i == 0 {
					indexName = "Main"
				}
				if indexName != "" {
					href := "/"
					if i != 0 {
						href = fmt.Sprintf("/?tab=%d", i)
					}
					lastSha := ""
					if len(t.Pages) > 0 {
						lastSha = t.Pages[len(t.Pages)-1].Sha()
					}
					tabs = append(tabs, TabInfo{Index: i, Name: t.Name, IndexName: indexName, Href: href, LastPageSha: lastSha})
				}
			}
			return tabs, nil
		},
		"tabName": func() string {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}

			bookmarks, _, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			var bookmark = defaultBookmarks
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					bookmark = ""
				} else {
					return ""
				}
			} else {
				bookmark = bookmarks
			}
			tabs := ParseBookmarks(bookmark)
			tabStr := r.URL.Query().Get("tab")
			idx, err := strconv.Atoi(tabStr)
			if err != nil || idx < 0 || idx >= len(tabs) {
				idx = 0
			}
			name := tabs[idx].DisplayName()
			if name == "" && idx == 0 {
				name = "Main"
			}
			return name
		},
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}

			bookmarks, _, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			var bookmark = defaultBookmarks
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					bookmark = ""
				} else {
					return nil, fmt.Errorf("bookmarkColumns: %w", err)
				}
			} else {
				bookmark = bookmarks
			}
			tabsData := ParseBookmarks(bookmark)
			var columns []*BookmarkColumn
			for _, t := range tabsData {
				for _, p := range t.Pages {
					for _, b := range p.Blocks {
						columns = append(columns, b.Columns...)
					}
				}
			}
			return columns, nil
		},
		"tags": func() ([]*Tag, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}

			tags, err := GetTags(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetTags: %w", err)
			}
			return tags, nil
		},
		"branches": func() ([]*Branch, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}
			branches, err := GetBranches(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetBranches: %w", err)
			}
			return branches, nil
		},
		"commits": func() ([]*Commit, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			ref := r.URL.Query().Get("ref")
			commits, err := GetCommits(r.Context(), login, token, ref, page, CommitsPerPage)
			if err != nil {
				return nil, fmt.Errorf("GetCommits: %w", err)
			}
			return commits, nil
		},
		"prevCommit": func() string {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}
			ref := r.URL.Query().Get("historyRef")
			sha := r.URL.Query().Get("ref")
			if ref == "" || sha == "" {
				return ""
			}
			prev, _, err := GetAdjacentCommits(r.Context(), login, token, ref, sha)
			if err != nil {
				return ""
			}
			return prev
		},
		"nextCommit": func() string {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil {
				login = githubUser.Login
			}
			ref := r.URL.Query().Get("historyRef")
			sha := r.URL.Query().Get("ref")
			if ref == "" || sha == "" {
				return ""
			}
			_, next, err := GetAdjacentCommits(r.Context(), login, token, ref, sha)
			if err != nil {
				return ""
			}
			return next
		},
		"isSearchURL": func(u string) bool {
			return strings.HasPrefix(u, "search:")
		},
		"searchURL": func(u string) string {
			return strings.TrimPrefix(u, "search:")
		},
	}
}

func Bookmarks(r *http.Request) (string, error) {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	bookmarks, _, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("bookmarks: %w", err)
	}
	return bookmarks, nil
}

func BookmarksExist(r *http.Request) (bool, error) {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	bookmarks, _, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("bookmarks exist: %w", err)
	}
	return bookmarks != "", nil
}

func errorMessage(code string) string {
	switch code {
	case "invalid":
		return "Invalid username or password"
	case "exists":
		return "Account already exists"
	default:
		return code
	}
}
