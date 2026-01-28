package gobookmarks

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
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

// ensureRepo checks for the bookmarks repository and creates it with
// some default content when missing.
func ensureRepo(ctx context.Context, p Provider, user string, token *oauth2.Token) error {
	log.Printf("checking repo for %s", user)

	cfg := ctx.Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()

	exists, err := p.RepoExists(ctx, user, token, repoName)
	if err != nil {
		log.Printf("repo check error: %v", err)
		return err
	}
	if !exists {
		log.Printf("creating repo %s for %s", repoName, user)
		if err := p.CreateRepo(ctx, user, token, repoName); err != nil {
			log.Printf("create repo: %v", err)
			return err
		}
	}

	b, _, err := p.GetBookmarks(ctx, user, "", token)
	if err != nil {
		log.Printf("get bookmarks: %v", err)
		return err
	}
	if b == "" {
		log.Printf("creating initial bookmarks for %s", user)
		if err := p.CreateBookmarks(ctx, user, token, "main", defaultBookmarks); err != nil {
			log.Printf("create bookmarks: %v", err)
			return err
		}
	}
	return nil
}

func LoginWithProvider(w http.ResponseWriter, r *http.Request) error {
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	providerName := mux.Vars(r)["provider"]
	p := GetProvider(providerName)
	if p == nil {
		http.NotFound(w, r)
		return nil
	}

	session, err := getSession(w, r, cfg)
	if session, err = sanitizeSession(w, r, session, err, cfg); err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	session.Values["Provider"] = providerName
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}

	creds := cfg.GetProviderCreds(providerName)
	if creds == nil {
		http.NotFound(w, r)
		return nil
	}
	oacfg := p.Config(r.Context(), creds.ID, creds.Secret, cfg.GetRedirectURL())
	if oacfg == nil {
		http.NotFound(w, r)
		return nil
	}
	http.Redirect(w, r, oacfg.AuthCodeURL(providerName), http.StatusTemporaryRedirect)
	return nil
}

func Oauth2CallbackPage(w http.ResponseWriter, r *http.Request) error {
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)

	type ErrorData struct {
		*CoreData
		Error string
	}

	session, err := getSession(w, r, cfg)
	if session, err = sanitizeSession(w, r, session, err, cfg); err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	providerName := r.URL.Query().Get("state")
	if providerName == "" {
		providerName, _ = session.Values["Provider"].(string)
	}
	p := GetProvider(providerName)
	if p == nil {
		return fmt.Errorf("unknown provider")
	}

	creds := cfg.GetProviderCreds(providerName)
	if creds == nil {
		return fmt.Errorf("provider does not support login")
	}
	oacfg := p.Config(r.Context(), creds.ID, creds.Secret, cfg.GetRedirectURL())
	if oacfg == nil {
		return fmt.Errorf("provider does not support login")
	}
	token, err := oacfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			status := 0
			if retrieveErr.Response != nil {
				status = retrieveErr.Response.StatusCode
			}

			if status == 0 || (status >= 400 && status < 500) {
				log.Printf("oauth exchange failed for %s: %v", providerName, err)
				session.Options.MaxAge = -1
				if saveErr := session.Save(r, w); saveErr != nil {
					log.Printf("session save error after oauth failure: %v", saveErr)
				}
				http.Redirect(w, r, fmt.Sprintf("/login/%s?error=oauth", providerName), http.StatusSeeOther)
				return ErrHandled
			}
		}
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
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	session, err := getSession(w, r, cfg)
	if session, err = sanitizeSession(w, r, session, err, cfg); err != nil {
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
	if err != nil {
		log.Printf("git login check error for %s: %v", user, err)
	}
	if err != nil || !okPass {
		if !okPass {
			log.Printf("git login failed for %s: invalid password", user)
		}
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
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	user := r.FormValue("username")
	pass := r.FormValue("password")
	prov := GetProvider("git")
	ph, ok := prov.(PasswordHandler)
	if !ok {
		return fmt.Errorf("password handler not available")
	}
	if err := ph.CreateUser(r.Context(), user, pass); err != nil {
		if errors.Is(err, ErrUserExists) {
			log.Printf("git signup for %s failed: user exists", user)
			http.Redirect(w, r, "/login/git?error=exists", http.StatusSeeOther)
			return nil
		}
		log.Printf("git signup create user error for %s: %v", user, err)
		return err
	}
	if exists, err := prov.RepoExists(r.Context(), user, nil, repoName); err == nil && !exists {
		if err := prov.CreateRepo(r.Context(), user, nil, repoName); err != nil {
			log.Printf("git signup create repo error for %s: %v", user, err)
			return err
		}
	} else if err != nil {
		log.Printf("git signup repo check error for %s: %v", user, err)
		return err
	}
	if err := prov.CreateBookmarks(r.Context(), user, nil, "main", defaultBookmarks); err != nil {
		log.Printf("git signup create sample bookmarks error for %s: %v", user, err)
		return fmt.Errorf("create sample bookmarks: %w", err)
	}
	return nil
}

func SqlLoginAction(w http.ResponseWriter, r *http.Request) error {
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	session, err := getSession(w, r, cfg)
	if session, err = sanitizeSession(w, r, session, err, cfg); err != nil {
		return fmt.Errorf("session error: %w", err)
	}
	user := r.FormValue("username")
	pass := r.FormValue("password")
	p := GetProvider("sql")
	ph, ok := p.(PasswordHandler)
	if !ok {
		return fmt.Errorf("password handler not available")
	}
	okPass, err := ph.CheckPassword(r.Context(), user, pass)
	if err != nil {
		log.Printf("sql login check error for %s: %v", user, err)
	}
	if err != nil || !okPass {
		if !okPass {
			log.Printf("sql login failed for %s: invalid password", user)
		}
		http.Redirect(w, r, "/login/sql?error=invalid", http.StatusSeeOther)
		return nil
	}
	session.Values["Provider"] = "sql"
	session.Values["GithubUser"] = &User{Login: user}
	session.Values["Token"] = nil
	session.Values["version"] = version
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

func SqlSignupAction(w http.ResponseWriter, r *http.Request) error {
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()
	user := r.FormValue("username")
	pass := r.FormValue("password")
	prov := GetProvider("sql")
	ph, ok := prov.(PasswordHandler)
	if !ok {
		return fmt.Errorf("password handler not available")
	}
	if err := ph.CreateUser(r.Context(), user, pass); err != nil {
		if errors.Is(err, ErrUserExists) {
			log.Printf("sql signup for %s failed: user exists", user)
			http.Redirect(w, r, "/login/sql?error=exists", http.StatusSeeOther)
			return nil
		}
		log.Printf("sql signup create user error for %s: %v", user, err)
		return err
	}
	if exists, err := prov.RepoExists(r.Context(), user, nil, repoName); err == nil && !exists {
		if err := prov.CreateRepo(r.Context(), user, nil, repoName); err != nil {
			log.Printf("sql signup create repo error for %s: %v", user, err)
			return err
		}
	} else if err != nil {
		log.Printf("sql signup repo check error for %s: %v", user, err)
		return err
	}
	if err := prov.CreateBookmarks(r.Context(), user, nil, "main", defaultBookmarks); err != nil {
		log.Printf("sql signup create sample bookmarks error for %s: %v", user, err)
		return fmt.Errorf("create sample bookmarks: %w", err)
	}
	return nil
}

func UserAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cfg := request.Context().Value(ContextValues("configuration")).(*Configuration)
		// Get the session.
		session, err := getSession(writer, request, cfg)
		if session, err = sanitizeSession(writer, request, session, err, cfg); err != nil {
			log.Printf("session error: %v", err)
		}

		ctx := context.WithValue(request.Context(), ContextValues("session"), session)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func getSession(w http.ResponseWriter, r *http.Request, cfg *Configuration) (*sessions.Session, error) {
	session, err := cfg.SessionStore.Get(r, cfg.SessionName)
	if err != nil {
		return nil, err
	}
	if v, ok := session.Values["version"].(string); !ok || v != version {
		session.Options.MaxAge = -1
		if err := session.Save(r, w); err != nil {
			return nil, err
		}
		session, err = cfg.SessionStore.New(r, cfg.SessionName)
		if err != nil {
			return nil, err
		}
		session.Values = make(map[interface{}]interface{})
		session.IsNew = true
	}
	return session, nil
}
