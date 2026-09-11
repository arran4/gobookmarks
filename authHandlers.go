package gobookmarks

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"strings"
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

	data.UserRef = ""

	return nil
}

var (
	SessionStore sessions.Store
)

// ensureRepo checks for the bookmarks repository and creates it with
// some default content when missing.
func ensureRepo(ctx context.Context, p Provider, user string, token *oauth2.Token) error {
	log.Printf("checking repo for %s", user)

	repoName := Config.GetRepoName()
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
	providerName := mux.Vars(r)["provider"]
	p := GetProvider(providerName)
	if p == nil {
		http.NotFound(w, r)
		return nil
	}

	session := GetSession(w, r)
	if session == nil {
		return fmt.Errorf("session error")
	}

	redirect := r.URL.Query().Get("redirect")
	if redirect == "/" {
		redirect = ""
	}

	session.Values["Provider"] = providerName
	delete(session.Values, "Redirect")
	if redirect != "" && len(redirect) < 2048 {
		session.Values["Redirect"] = redirect
	}
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}

	creds := providerCreds(providerName)
	if creds == nil {
		http.NotFound(w, r)
		return ErrHandled
	}
	cfg := p.Config(creds.ID, creds.Secret, Config.GetOauthRedirectURL())
	if cfg == nil {
		http.NotFound(w, r)
		return ErrHandled
	}

	stateNonceBytes := make([]byte, 16)
	if _, err := rand.Read(stateNonceBytes); err != nil {
		return fmt.Errorf("failed to generate state nonce: %w", err)
	}
	stateNonce := hex.EncodeToString(stateNonceBytes)
	session.Values["OauthState"] = stateNonce
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save oauth state: %w", err)
	}

	state := providerName + ":" + stateNonce
	if redirect != "" && len(redirect) < 2048 {
		state = providerName + ":" + stateNonce + ":" + redirect
	}

	http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusTemporaryRedirect)
	return ErrHandled
}

func Oauth2CallbackPage(w http.ResponseWriter, r *http.Request) error {

	session := GetSession(w, r)
	if session == nil {
		return fmt.Errorf("session error")
	}

	var providerName, stateNonce, redirectUrl string
	stateParam := r.URL.Query().Get("state")
	if stateParam != "" {
		parts := strings.SplitN(stateParam, ":", 3)
		providerName = parts[0]
		if len(parts) > 1 {
			stateNonce = parts[1]
		}
		if len(parts) > 2 && parts[2] != "" && len(parts[2]) < 2048 {
			redirectUrl = parts[2]
		}
	} else {
		providerName, _ = session.Values["Provider"].(string)
	}

	expectedNonce, _ := session.Values["OauthState"].(string)
	if stateNonce == "" || expectedNonce == "" || subtle.ConstantTimeCompare([]byte(stateNonce), []byte(expectedNonce)) != 1 {
		return fmt.Errorf("invalid oauth state parameter")
	}

	// Consume the nonce and prepare the session for authenticated state
	delete(session.Values, "OauthState")
	if redirectUrl != "" {
		session.Values["Redirect"] = redirectUrl
	}
	p := GetProvider(providerName)
	if p == nil {
		return fmt.Errorf("unknown provider")
	}

	creds := providerCreds(providerName)
	if creds == nil {
		return fmt.Errorf("provider does not support login")
	}
	cfg := p.Config(creds.ID, creds.Secret, Config.GetOauthRedirectURL())
	if cfg == nil {
		return fmt.Errorf("provider does not support login")
	}
	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
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
	session := GetSession(w, r)
	if session == nil {
		return fmt.Errorf("session error")
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
		redirectURL := "/login/git?error=invalid"
		if r.FormValue("redirect") != "" {
			redirectURL += "&redirect=" + url.QueryEscape(r.FormValue("redirect"))
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return ErrHandled
	}
	session.Values["Provider"] = "git"
	session.Values["GithubUser"] = &User{Login: user}
	session.Values["Token"] = nil
	session.Values["version"] = version
	redirect := r.FormValue("redirect")
	delete(session.Values, "Redirect")
	if redirect != "" && len(redirect) < 2048 {
		session.Values["Redirect"] = redirect
	}
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
			log.Printf("git signup for %s failed: user exists", user)
			redirectURL := "/login/git?error=exists"
			if r.FormValue("redirect") != "" {
				redirectURL += "&redirect=" + url.QueryEscape(r.FormValue("redirect"))
			}
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return ErrHandled
		}
		log.Printf("git signup create user error for %s: %v", user, err)
		return err
	}
	repoName := Config.GetRepoName()
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
	redirectURL := "/login/git"
	if r.FormValue("redirect") != "" {
		redirectURL += "?redirect=" + url.QueryEscape(r.FormValue("redirect"))
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return ErrHandled
}

func SqlLoginAction(w http.ResponseWriter, r *http.Request) error {
	session := GetSession(w, r)
	if session == nil {
		return fmt.Errorf("session error")
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
		redirectURL := "/login/sql?error=invalid"
		if r.FormValue("redirect") != "" {
			redirectURL += "&redirect=" + url.QueryEscape(r.FormValue("redirect"))
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return ErrHandled
	}
	session.Values["Provider"] = "sql"
	session.Values["GithubUser"] = &User{Login: user}
	session.Values["Token"] = nil
	session.Values["version"] = version
	redirect := r.FormValue("redirect")
	delete(session.Values, "Redirect")
	if redirect != "" && len(redirect) < 2048 {
		session.Values["Redirect"] = redirect
	}
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("session save: %w", err)
	}
	return nil
}

func SqlSignupAction(w http.ResponseWriter, r *http.Request) error {
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
			redirectURL := "/login/sql?error=exists"
			if r.FormValue("redirect") != "" {
				redirectURL += "&redirect=" + url.QueryEscape(r.FormValue("redirect"))
			}
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return ErrHandled
		}
		log.Printf("sql signup create user error for %s: %v", user, err)
		return err
	}
	repoName := Config.GetRepoName()
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
	redirectURL := "/login/sql"
	if r.FormValue("redirect") != "" {
		redirectURL += "?redirect=" + url.QueryEscape(r.FormValue("redirect"))
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return ErrHandled
}

func UserAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Get the session.
		session := GetSession(writer, request)
		if session == nil {
			log.Printf("session error")
		}

		ctx := context.WithValue(request.Context(), ContextValues("session"), session)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func GetSession(w http.ResponseWriter, r *http.Request) *sessions.Session {
	// Prefer context session to guarantee exactly one object per request
	if r != nil {
		if ctxSess, ok := r.Context().Value(ContextValues("session")).(*sessions.Session); ok && ctxSess != nil {
			return ctxSess
		}
	}

	session, err := SessionStore.Get(r, Config.GetSessionName())

	if r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") {
		session.Options.Secure = true
	} else {
		session.Options.Secure = false
	}

	session, err = sanitizeSession(w, r, session, err)
	if err != nil {
		log.Printf("session error: %v", err)
	}

	// Validate version only if the session claims to be authenticated
	if session.Values["GithubUser"] != nil {
		if v, ok := session.Values["version"].(string); !ok || v != version {
			// Invalidate and rotate
			session.Options.MaxAge = -1
			if w != nil {
				if saveErr := session.Save(r, w); saveErr != nil {
					log.Printf("failed to clear old session: %v", saveErr)
				}
			}

			// Create a brand new session object in-place for subsequent uses
			session, _ = SessionStore.New(r, Config.GetSessionName())
			if r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") {
				session.Options.Secure = true
			} else {
				session.Options.Secure = false
			}
			session.Values = make(map[interface{}]interface{})
			session.IsNew = true
		}
	}

	return session
}
