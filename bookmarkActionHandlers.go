package a4webbm

import (
	"database/sql"
	"github.com/gorilla/sessions"
	"net/http"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	queries := r.Context().Value(ContextValues("queries")).(*Queries)
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	userRef, _ := session.Values["UserRef"].(string)

	if err := queries.UpdateBookmarks(r.Context(), UpdateBookmarksParams{
		List: sql.NullString{
			String: text,
			Valid:  true,
		},
		Userreference: userRef,
	}); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	queries := r.Context().Value(ContextValues("queries")).(*Queries)
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	userRef, _ := session.Values["UserRef"].(string)

	if err := queries.CreateBookmarks(r.Context(), CreateBookmarksParams{
		List: sql.NullString{
			String: text,
			Valid:  true,
		},
		Userreference: userRef,
	}); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}
}
