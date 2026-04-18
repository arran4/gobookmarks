package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
)

// EnableCSSColumnsAction stores a session flag to use CSS column layout.
func EnableCSSColumnsAction(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["useCSSColumns"] = true
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

// DisableCSSColumnsAction stores a session flag to use table layout.
func DisableCSSColumnsAction(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	session.Values["useCSSColumns"] = false
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}
