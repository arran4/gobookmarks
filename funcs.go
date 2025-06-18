package gobookmarks

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"strings"
	"time"
)

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
		"add1": func(i int) int {
			return i + 1
		},
		"useCssColumns": func() bool {
			return UseCssColumns
		},
		"showFooter": func() bool {
			return !NoFooter
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
			tabName := r.URL.Query().Get("tab")
			var pages []*BookmarkPage
			if tabName != "" {
				for _, t := range tabs {
					if t.Name == tabName {
						pages = t.Pages
						break
					}
				}
			} else {
				for _, t := range tabs {
					if t.Name == "" {
						pages = t.Pages
						break
					}
				}
				if pages == nil && len(tabs) > 0 {
					pages = tabs[0].Pages
				}
			}
			return pages, nil
		},
		"bookmarkTabs": func() ([]string, error) {
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
			var tabs []string
			for i, t := range tabsData {
				name := t.DisplayName()
				if name == "" && i == 0 {
					name = "Main"
				}
				if name != "" {
					tabs = append(tabs, name)
				}
			}
			return tabs, nil
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
						if b.HR {
							continue
						}
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
			commits, err := GetCommits(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetCommits: %w", err)
			}
			return commits, nil
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
