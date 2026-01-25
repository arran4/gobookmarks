package gobookmarks

import (
	"context"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

func init() {
	gob.Register(&User{})
	gob.Register(&oauth2.Token{})
}

type CoreData struct {
	Title       string
	AutoRefresh bool
	UserRef     string
	EditMode    bool
	Tab         int
}

type Configuration struct {
	Config
	OauthRedirectURL string
	SessionStore     sessions.Store
	SessionName      string
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func (c *Configuration) CoreAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		session := request.Context().Value(ContextValues("session")).(*sessions.Session)
		githubUser, _ := session.Values["GithubUser"].(*User)
		providerName, _ := session.Values["Provider"].(string)

		login := ""
		if githubUser != nil {
			login = githubUser.Login
		}

		title := c.Title
		if title == "" {
			title = "gobookmarks"
		}
		if version == "dev" && !strings.HasPrefix(title, "dev: ") {
			title = "dev: " + title
		}

		ctx := context.WithValue(request.Context(), ContextValues("provider"), providerName)
		ctx = context.WithValue(ctx, ContextValues("configuration"), c)

		editMode := request.URL.Query().Get("edit") == "1"
		tab := TabFromRequest(request)
		ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{
			UserRef:  login,
			Title:    title,
			EditMode: editMode,
			Tab:      tab,
		})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (c *Configuration) GetProviderCreds(name string) *ProviderCreds {
	switch name {
	case "github":
		if c.GithubClientID == "" || c.GithubSecret == "" {
			return nil
		}
		return &ProviderCreds{ID: c.GithubClientID, Secret: c.GithubSecret}
	case "gitlab":
		if c.GitlabClientID == "" || c.GitlabSecret == "" {
			return nil
		}
		return &ProviderCreds{ID: c.GitlabClientID, Secret: c.GitlabSecret}
	case "git":
		if c.LocalGitPath == "" {
			return nil
		}
		return &ProviderCreds{}
	case "sql":
		if c.DBConnectionProvider == "" {
			return nil
		}
		return &ProviderCreds{}
	default:
		return nil
	}
}

type ContextValues string
