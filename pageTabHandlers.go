package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func MoveTabHandler(w http.ResponseWriter, r *http.Request) error {
	idxStr := r.URL.Query().Get("index")
	dir := r.URL.Query().Get("dir")
	idx, err := strconv.Atoi(idxStr)
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
	if idx < 0 || idx >= len(tabs) {
		return fmt.Errorf("tab index out of range")
	}
	tabs = moveTab(tabs, idx, delta)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func MovePageHandler(w http.ResponseWriter, r *http.Request) error {
	tabIdxStr := r.URL.Query().Get("tab")
	idxStr := r.URL.Query().Get("index")
	dir := r.URL.Query().Get("dir")
	tIdx, err := strconv.Atoi(tabIdxStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	idx, err := strconv.Atoi(idxStr)
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
	if tIdx < 0 || tIdx >= len(tabs) {
		return fmt.Errorf("tab index out of range")
	}
	movePage(tabs[tIdx], idx, delta)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func AddPageHandler(w http.ResponseWriter, r *http.Request) error {
	tabIdxStr := r.URL.Query().Get("tab")
	idxStr := r.URL.Query().Get("index")
	tIdx, err := strconv.Atoi(tabIdxStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	idx, err := strconv.Atoi(idxStr)
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
	if tIdx < 0 || tIdx >= len(tabs) {
		return fmt.Errorf("tab index out of range")
	}
	insertPage(tabs[tIdx], idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func DeletePageHandler(w http.ResponseWriter, r *http.Request) error {
	tabIdxStr := r.URL.Query().Get("tab")
	idxStr := r.URL.Query().Get("index")
	tIdx, err := strconv.Atoi(tabIdxStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	idx, err := strconv.Atoi(idxStr)
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
	if tIdx < 0 || tIdx >= len(tabs) {
		return fmt.Errorf("tab index out of range")
	}
	deletePage(tabs[tIdx], idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func AddTabHandler(w http.ResponseWriter, r *http.Request) error {
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
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
	tabs = insertTab(tabs, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func DeleteTabHandler(w http.ResponseWriter, r *http.Request) error {
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
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
	tabs = deleteTab(tabs, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}
