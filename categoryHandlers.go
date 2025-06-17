package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func MoveCategoryHandler(w http.ResponseWriter, r *http.Request) error {
	tStr := r.URL.Query().Get("tab")
	pStr := r.URL.Query().Get("page")
	bStr := r.URL.Query().Get("block")
	cStr := r.URL.Query().Get("col")
	iStr := r.URL.Query().Get("index")
	dir := r.URL.Query().Get("dir")
	ret := r.URL.Query().Get("return")

	tIdx, err := strconv.Atoi(tStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	pIdx, err := strconv.Atoi(pStr)
	if err != nil {
		return fmt.Errorf("invalid page index: %w", err)
	}
	bIdx, err := strconv.Atoi(bStr)
	if err != nil {
		return fmt.Errorf("invalid block index: %w", err)
	}
	colIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid column index: %w", err)
	}
	idx, err := strconv.Atoi(iStr)
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
	moveCategory(tabs[tIdx], pIdx, bIdx, colIdx, idx, delta)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	if ret != "" {
		http.Redirect(w, r, ret, http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
	return ErrHandled
}

func AddCategoryHandler(w http.ResponseWriter, r *http.Request) error {
	tStr := r.URL.Query().Get("tab")
	pStr := r.URL.Query().Get("page")
	bStr := r.URL.Query().Get("block")
	cStr := r.URL.Query().Get("col")
	iStr := r.URL.Query().Get("index")

	tIdx, err := strconv.Atoi(tStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	pIdx, err := strconv.Atoi(pStr)
	if err != nil {
		return fmt.Errorf("invalid page index: %w", err)
	}
	bIdx, err := strconv.Atoi(bStr)
	if err != nil {
		return fmt.Errorf("invalid block index: %w", err)
	}
	colIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid column index: %w", err)
	}
	idx, err := strconv.Atoi(iStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	ret := r.URL.Query().Get("return")

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
	insertCategory(tabs[tIdx], pIdx, bIdx, colIdx, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	if ret != "" {
		http.Redirect(w, r, ret, http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
	return ErrHandled
}

func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) error {
	tStr := r.URL.Query().Get("tab")
	pStr := r.URL.Query().Get("page")
	bStr := r.URL.Query().Get("block")
	cStr := r.URL.Query().Get("col")
	iStr := r.URL.Query().Get("index")

	tIdx, err := strconv.Atoi(tStr)
	if err != nil {
		return fmt.Errorf("invalid tab index: %w", err)
	}
	pIdx, err := strconv.Atoi(pStr)
	if err != nil {
		return fmt.Errorf("invalid page index: %w", err)
	}
	bIdx, err := strconv.Atoi(bStr)
	if err != nil {
		return fmt.Errorf("invalid block index: %w", err)
	}
	colIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid column index: %w", err)
	}
	idx, err := strconv.Atoi(iStr)
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
	deleteCategory(tabs[tIdx], pIdx, bIdx, colIdx, idx)
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, sha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}
