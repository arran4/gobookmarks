package a4webbm

import (
	"github.com/google/go-github/v55/github"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	if err := UpdateBookmarks(r.Context(), login, token, text); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*github.User)
	token, _ := session.Values["Token"].(*oauth2.Token)

	login := ""
	if githubUser != nil && githubUser.Login != nil {
		login = *githubUser.Login
	}

	if err := CreateBookmarks(r.Context(), login, token, text); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}
