package a4webbm

import (
	"fmt"
	"github.com/google/go-github/v55/github"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	if err := UpdateBookmarks(r.Context(), login, token, branch, text); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	if err := CreateBookmarks(r.Context(), login, token, branch, text); err != nil {
		return fmt.Errorf("crateBookmark error: %w", err)
	}
	return nil
}
