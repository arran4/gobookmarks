package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
)

// EnableCssColumnsAction stores a session flag to use CSS column layout.
func EnableCssColumnsAction(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["useCssColumns"] = true
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

// DisableCssColumnsAction stores a session flag to use table layout.
func DisableCssColumnsAction(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["useCssColumns"] = false
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}
