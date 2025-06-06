package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	login, _ := session.Values["UserLogin"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")

	_, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, text, curSha); err != nil {

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, text, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	login, _ := session.Values["UserLogin"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")

	if err := CreateBookmarks(r.Context(), login, token, branch, text); err != nil {
		return fmt.Errorf("crateBookmark error: %w", err)
	}
	return nil
}

func CategoryEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	login, _ := session.Values["UserLogin"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")

	currentBookmarks, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}
	updated, err := ReplaceCategoryByIndex(currentBookmarks, idx, text)
	if err != nil {
		return fmt.Errorf("ReplaceCategory: %w", err)
	}
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func CategoryEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	currentBookmarks, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}
	updated, err := ReplaceCategoryByIndex(currentBookmarks, idx, text)
	if err != nil {
		return fmt.Errorf("ReplaceCategory: %w", err)
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}
