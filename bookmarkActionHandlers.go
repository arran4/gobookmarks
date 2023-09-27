package a4webbm

import (
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)

	if err := UpdateBookmarks(r.Context(), githubUser, token, text); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)

	if err := CreateBookmarks(r.Context(), githubUser, token, text); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}
