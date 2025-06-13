package gobookmarks

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
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

// ensureRepo checks for the bookmarks repository and creates it with
// some default content when missing.
func ensureRepo(ctx context.Context, p Provider, user string, token *oauth2.Token) error {
	if err := p.CreateBookmarks(ctx, user, token, "main", defaultBookmarks); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			if err := p.CreateRepo(ctx, user, token, RepoName); err != nil {
				return err
			}
			return p.CreateBookmarks(ctx, user, token, "main", defaultBookmarks)
		}
		return err
	}
	return nil
}

func LoginWithProvider(w http.ResponseWriter, r *http.Request) error {
	providerName := mux.Vars(r)["provider"]
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

	creds := providerCreds(providerName)
	if creds == nil {
		http.NotFound(w, r)
		return nil
	}
	cfg := p.Config(creds.ID, creds.Secret, OauthRedirectURL)
	if cfg == nil {
		http.NotFound(w, r)
		return nil
	}
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

	creds := providerCreds(providerName)
	if creds == nil {
		return fmt.Errorf("provider does not support login")
	}
	cfg := p.Config(creds.ID, creds.Secret, OauthRedirectURL)
	if cfg == nil {
		return fmt.Errorf("provider does not support login")
	}
	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return fmt.Errorf("exchange error: %w", err)
	}

	user, err := p.CurrentUser(r.Context(), token)
	if err != nil {
		return fmt.Errorf("user lookup error: %w", err)
	}

	if err := ensureRepo(r.Context(), p, user.Login, token); err != nil {
		// expire the session from the login step
		session.Options.MaxAge = -1
		_ = session.Save(r, w)
		return fmt.Errorf("repository setup failed: %w", err)
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

func GitLoginAction(w http.ResponseWriter, r *http.Request) error {
	session, err := getSession(w, r)
	if err != nil {
		return fmt.Errorf("session error: %w", err)
	}
	user := r.FormValue("username")
	pass := r.FormValue("password")
	p := GetProvider("git")
	ph, ok := p.(PasswordHandler)
	if !ok {
		return fmt.Errorf("password handler not available")
	}
	okPass, err := ph.CheckPassword(r.Context(), user, pass)
	if err != nil || !okPass {
		http.Redirect(w, r, "/login/git?error=invalid", http.StatusSeeOther)
		return nil
	}
	session.Values["Provider"] = "git"
	session.Values["GithubUser"] = &User{Login: user}
	session.Values["Token"] = nil
	session.Values["version"] = version
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

func GitSignupAction(w http.ResponseWriter, r *http.Request) error {
	user := r.FormValue("username")
	pass := r.FormValue("password")
	prov := GetProvider("git")
	ph, ok := prov.(PasswordHandler)
	if !ok {
		return fmt.Errorf("password handler not available")
	}
	if err := ph.CreateUser(r.Context(), user, pass); err != nil {
		if errors.Is(err, ErrUserExists) {
			http.Redirect(w, r, "/login/git?error=exists", http.StatusSeeOther)
			return nil
		}
		return err
	}
	if err := prov.CreateRepo(r.Context(), user, nil, RepoName); err != nil {
		return err
	}
	if err := prov.CreateBookmarks(r.Context(), user, nil, "main", defaultBookmarks); err != nil {
		return fmt.Errorf("create sample bookmarks: %w", err)
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
