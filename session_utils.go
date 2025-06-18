package gobookmarks

import (
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

// sanitizeSession returns a new session when err indicates the cookie was
// invalid. The original session is cleared so the client replaces it.
func sanitizeSession(w http.ResponseWriter, r *http.Request, session *sessions.Session, err error) (*sessions.Session, error) {
	if err == nil {
		return session, nil
	}
	scErr := new(securecookie.MultiError)
	if (errors.As(err, scErr) && scErr.IsDecode() && !scErr.IsInternal() && !scErr.IsUsage()) || errors.Is(err, securecookie.ErrMacInvalid) {
		log.Printf("session error: %v", err)
		if session != nil {
			session.Options.MaxAge = -1
			if saveErr := session.Save(r, w); saveErr != nil {
				log.Printf("session clear error: %v", saveErr)
			}
		}
		session, _ = SessionStore.New(r, SessionName)
		return session, nil
	}
	return session, err
}
