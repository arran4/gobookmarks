package gobookmarks

import (
	"github.com/gorilla/sessions"
	"net/http"
)

func StartEditMode(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["EditMode"] = true
	if err := session.Save(r, w); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}

func StopEditMode(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	delete(session.Values, "EditMode")
	if err := session.Save(r, w); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return ErrHandled
}
