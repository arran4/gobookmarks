package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
)

// StartEditMode enables edit mode by storing a flag in the user's session.
func StartEditMode(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["editMode"] = true
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

// StopEditMode disables edit mode and clears the flag from the session.
func StopEditMode(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	delete(session.Values, "editMode")
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}
