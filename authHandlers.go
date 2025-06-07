package gobookmarks

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

func UserLogoutAction(w http.ResponseWriter, r *http.Request) error {
	type Data struct {
		*CoreData
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
	}

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	delete(session.Values, "GithubUser")
	delete(session.Values, "Token")

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session.Save Error: %w", err)
	}

	data.CoreData.UserRef = ""

	return nil
}

var (
	Oauth2Config *oauth2.Config
	SessionStore sessions.Store
	SessionName  string
)

func Oauth2CallbackPage(w http.ResponseWriter, r *http.Request) error {

	type ErrorData struct {
		*CoreData
		Error string
	}

	token, err := Oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return fmt.Errorf("exchange error: %w", err)
	}

	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	user, err := ActiveProvider.CurrentUser(r.Context(), token)
	if err != nil {
		return fmt.Errorf("user lookup error: %w", err)
	}

	session.Values["GithubUser"] = user
	session.Values["Token"] = token

	if err := session.Save(r, w); err != nil {
		log.Printf("Exchange error: %s", err)
		return fmt.Errorf("exchange error: %w", err)
	}

	return nil
}

func UserAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Get the session.
		session, err := SessionStore.Get(request, SessionName)
		if err != nil {
			// ignore common decode errors
			if scErr := new(securecookie.MultiError); !(errors.As(err, scErr) && scErr.IsDecode() && !scErr.IsInternal() && !scErr.IsUsage()) &&
				!errors.Is(err, securecookie.ErrMacInvalid) {
				log.Printf("session error: %v", err)
			}
			if session != nil {
				// invalidate the existing cookie
				session.Options.MaxAge = -1
				if saveErr := session.Save(request, writer); saveErr != nil {
					log.Printf("session clear error: %v", saveErr)
				}
			}
			// start with a fresh session so the request still succeeds
			session, _ = SessionStore.New(request, SessionName)
		}

		ctx := context.WithValue(request.Context(), ContextValues("session"), session)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
