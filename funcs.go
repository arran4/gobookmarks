package a4webbm

import (
	"database/sql"
	"errors"
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
			userRef, _ := session.Values["UserRef"].(string)
			return userRef != "", nil
		},
		"bookmarks": func() (string, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)

			bookmarks, err := GetBookmarksForUser(r.Context(), githubUser, "", token)
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return defaultBookmarks, nil
				default:
					return "", err
				}
			}
			return bookmarks, nil
		},
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			githubUser, _ := session.Values["GithubUser"].(string)
			token, _ := session.Values["Token"].(*oauth2.Token)

			bookmarks, err := GetBookmarksForUser(r.Context(), githubUser, "", token)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
				default:
					return nil, err
				}
			} else {
				bookmarkString = bookmarks
			}
			return PreprocessBookmarks(bookmarkString), nil
		},
	}
}
