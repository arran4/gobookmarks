package gobookmarks

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
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
	delete(session.Values, "Provider")

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session.Save Error: %w", err)
	}

	data.CoreData.UserRef = ""

	return nil
}

var (
	SessionStore sessions.Store
	SessionName  string
)

func LoginWithProvider(w http.ResponseWriter, r *http.Request) error {
	providerName := r.PathValue("provider")
	p := GetProvider(providerName)
	if p == nil {
		http.NotFound(w, r)
		return nil
	}

	session, err := getSession(w, r)
	if err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	session.Values["Provider"] = providerName
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}

	id, secret := providerCreds(providerName)
	cfg := p.OAuth2Config(id, secret, OauthRedirectURL)
	http.Redirect(w, r, cfg.AuthCodeURL(""), http.StatusTemporaryRedirect)
	return nil
}

func Oauth2CallbackPage(w http.ResponseWriter, r *http.Request) error {

	type ErrorData struct {
		*CoreData
		Error string
	}

	session, err := getSession(w, r)
	if err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	providerName, _ := session.Values["Provider"].(string)
	p := GetProvider(providerName)
	if p == nil {
		return fmt.Errorf("unknown provider")
	}

	id, secret := providerCreds(providerName)
	cfg := p.OAuth2Config(id, secret, OauthRedirectURL)
	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return fmt.Errorf("exchange error: %w", err)
	}

	user, err := p.CurrentUser(r.Context(), token)
	if err != nil {
		return fmt.Errorf("user lookup error: %w", err)
	}

	session.Values["Provider"] = providerName
	session.Values["GithubUser"] = user
	session.Values["Token"] = token
	session.Values["version"] = version

	if err := session.Save(r, w); err != nil {
		log.Printf("Exchange error: %s", err)
		return fmt.Errorf("exchange error: %w", err)
	}

	return nil
}

func UserAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Get the session.
		session, err := getSession(writer, request)
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

func getSession(w http.ResponseWriter, r *http.Request) (*sessions.Session, error) {
	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		return nil, err
	}
	if v, ok := session.Values["version"].(string); !ok || v != version {
		session.Options.MaxAge = -1
		if err := session.Save(r, w); err != nil {
			return nil, err
		}
		session, err = SessionStore.New(r, SessionName)
		if err != nil {
			return nil, err
		}
		session.Values = make(map[interface{}]interface{})
		session.IsNew = true
	}
	return session, nil
}
