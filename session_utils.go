package gobookmarks

import (
	"crypto/sha256"
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

// InitSessionStore creates a new CookieStore with encrypted sessions
func InitSessionStore(key []byte) *sessions.CookieStore {
	hashKey := sha256.Sum256(append([]byte("hash-"), key...))
	blockKey := sha256.Sum256(append([]byte("block-"), key...))

	store := sessions.NewCookieStore(hashKey[:], blockKey[:], key, nil)

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	}

	return store
}

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
			if w != nil {
				if saveErr := session.Save(r, w); saveErr != nil {
					log.Printf("session clear error: %v", saveErr)
				}
			}
		}
		session, _ = SessionStore.New(r, Config.GetSessionName())
		return session, nil
	}
	return session, err
}
