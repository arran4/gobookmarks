package a4webbm

import (
	"fmt"
	"github.com/google/go-github/v55/github"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"strings"
	"time"
)

var (
	defaultBookmarks = "Category: Example 1\nhttp://www.google.com.au Google\nColumn\nCategory: Example 2\nhttp://www.google.com.au Google\nhttp://www.google.com.au Google\n"
)

func NewFuncs(r *http.Request) template.FuncMap {
	return map[string]any{
		"now": func() time.Time { return time.Now() },
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
		"loggedIn": func() (bool, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, ok := session.Values["GithubUser"].(*github.User)
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
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*github.User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil && githubUser.Login != nil {
				login = *githubUser.Login
			}

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
		"tags": func() ([]*github.RepositoryTag, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*github.User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil && githubUser.Login != nil {
				login = *githubUser.Login
			}

			tags, err := GetTags(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetTags: %w", err)
			}
			return tags, nil
		},
		"branches": func() ([]*github.Branch, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(*github.User)
			token, _ := session.Values["Token"].(*oauth2.Token)

			login := ""
			if githubUser != nil && githubUser.Login != nil {
				login = *githubUser.Login
			}
			branches, err := GetBranches(r.Context(), login, token)
			if err != nil {
				return nil, fmt.Errorf("GetBranches: %w", err)
			}
			return branches, nil
		},
	}
}

func Bookmarks(r *http.Request) (string, error) {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	bookmarks, err := GetBookmarks(r.Context(), login, "", token)
	if err != nil {
		return "", fmt.Errorf("bookmarks: %w", err)
	}
	return bookmarks, nil
}
