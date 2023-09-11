package a4webbm

import (
	"database/sql"
	"errors"
	"github.com/gorilla/sessions"
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
		"bookmarks": func() (string, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return defaultBookmarks, nil
				default:
					return "", err
				}
			}
			return bookmarks.List.String, nil
		},
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
				default:
					return nil, err
				}
			} else {
				bookmarkString = bookmarks.List.String
			}
			return PreprocessBookmarks(bookmarkString), nil
		},
	}
}
