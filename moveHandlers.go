package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func MoveTabAction(w http.ResponseWriter, r *http.Request) error {
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	to, _ := strconv.Atoi(r.URL.Query().Get("to"))
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "refs/heads/main"
	}
	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	list := ParseBookmarks(bookmarks)
	list.MoveTab(from, to)
	if err := UpdateBookmarks(r.Context(), login, token, ref, "main", list.String(), sha); err != nil {
		return fmt.Errorf("updateBookmarks: %w", err)
	}
	return nil
}

func MovePageAction(w http.ResponseWriter, r *http.Request) error {
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	to, _ := strconv.Atoi(r.URL.Query().Get("to"))
	tabName := r.URL.Query().Get("tab")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "refs/heads/main"
	}
	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	list := ParseBookmarks(bookmarks)
	for _, t := range list {
		if t.DisplayName() == tabName || t.Name == tabName {
			t.MovePage(from, to)
			break
		}
	}
	if err := UpdateBookmarks(r.Context(), login, token, ref, "main", list.String(), sha); err != nil {
		return fmt.Errorf("updateBookmarks: %w", err)
	}
	return nil
}

func MoveEntryAction(w http.ResponseWriter, r *http.Request) error {
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	to, _ := strconv.Atoi(r.URL.Query().Get("to"))
	catIdx, _ := strconv.Atoi(r.URL.Query().Get("category"))
	tabName := r.URL.Query().Get("tab")
	pageIdx, _ := strconv.Atoi(r.URL.Query().Get("page"))
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "refs/heads/main"
	}
	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	list := ParseBookmarks(bookmarks)
	for _, t := range list {
		if t.DisplayName() == tabName || t.Name == tabName {
			if pageIdx < len(t.Pages) {
				page := t.Pages[pageIdx]
				for _, blk := range page.Blocks {
					for _, col := range blk.Columns {
						for _, c := range col.Categories {
							if c.Index == catIdx {
								c.MoveEntry(from, to)
								break
							}
						}
					}
				}
			}
			break
		}
	}
	if err := UpdateBookmarks(r.Context(), login, token, ref, "main", list.String(), sha); err != nil {
		return fmt.Errorf("updateBookmarks: %w", err)
	}
	return nil
}
