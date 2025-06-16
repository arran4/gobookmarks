package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func AddEntryHandler(w http.ResponseWriter, r *http.Request) error {
	cStr := r.URL.Query().Get("cat")
	eStr := r.URL.Query().Get("index")
	cIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid category index: %w", err)
	}
	idx, err := strconv.Atoi(eStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")
	branch := r.URL.Query().Get("branch")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	current, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	tabs := PreprocessBookmarks(current)
	var cat *BookmarkCategory
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						if c.Index == cIdx {
							cat = c
							break
						}
					}
				}
			}
		}
	}
	if cat == nil {
		return fmt.Errorf("category not found")
	}
	insertEntry(cat, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/editEntry?cat=%d&entry=%d&ref=%s", cIdx, idx, r.URL.Query().Get("ref")), http.StatusTemporaryRedirect)
	return ErrHandled
}

func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) error {
	cStr := r.URL.Query().Get("cat")
	eStr := r.URL.Query().Get("index")
	cIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid category index: %w", err)
	}
	idx, err := strconv.Atoi(eStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")
	branch := r.URL.Query().Get("branch")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	current, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	tabs := PreprocessBookmarks(current)
	var cat *BookmarkCategory
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						if c.Index == cIdx {
							cat = c
							break
						}
					}
				}
			}
		}
	}
	if cat == nil {
		return fmt.Errorf("category not found")
	}
	deleteEntry(cat, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func MoveEntryHandler(w http.ResponseWriter, r *http.Request) error {
	cStr := r.URL.Query().Get("cat")
	eStr := r.URL.Query().Get("index")
	dir := r.URL.Query().Get("dir")
	cIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid category index: %w", err)
	}
	idx, err := strconv.Atoi(eStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	delta := 0
	if dir == "up" {
		delta = -1
	} else if dir == "down" {
		delta = 1
	} else {
		return fmt.Errorf("invalid dir")
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")
	branch := r.URL.Query().Get("branch")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	current, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	tabs := PreprocessBookmarks(current)
	var cat *BookmarkCategory
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						if c.Index == cIdx {
							cat = c
							break
						}
					}
				}
			}
		}
	}
	if cat == nil {
		return fmt.Errorf("category not found")
	}
	moveEntry(cat, idx, delta)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}
