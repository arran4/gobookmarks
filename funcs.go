package gobookmarks

import (
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
			return fmt.Sprintf("%s, commit %s, built at %s", version, commit, date)
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
		"OAuth2URL": func() string {
			return Oauth2Config.AuthCodeURL("")
		},
		"ref": func() string {
			return r.URL.Query().Get("ref")
		},
		"loggedIn": func() (bool, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			login, ok := session.Values["UserLogin"].(string)
			return ok && login != "", nil
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
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			login, _ := session.Values["UserLogin"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)

			bookmarks, err := GetBookmarks(r.Context(), login, r.URL.Query().Get("ref"), token)
			var bookmark = defaultBookmarks
			if err != nil {
				// TODO check for error type and if it's not exist, fall through
				return nil, fmt.Errorf("bookmarkColumns: %w", err)
			} else {
				bookmark = bookmarks
			}
			return PreprocessBookmarks(bookmark), nil
		},
		"tags": func() ([]*RepositoryTag, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			login, _ := session.Values["UserLogin"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)

			tags, err := GetTags(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetTags: %w", err)
			}
			return tags, nil
		},
		"branches": func() ([]*Branch, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			login, _ := session.Values["UserLogin"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)
			branches, err := GetBranches(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetBranches: %w", err)
			}
			return branches, nil
		},
		"commits": func() ([]*RepositoryCommit, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			login, _ := session.Values["UserLogin"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)
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
	login, _ := session.Values["UserLogin"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	bookmarks, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return "", fmt.Errorf("bookmarks: %w", err)
	}
	return bookmarks, nil
}
