package gobookmarks

import (
	"bufio"
	"context"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"strings"
)

var AppConfig = NewConfiguration()

func init() {
	gob.Register(&User{})
	gob.Register(&oauth2.Token{})
}

func CoreAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		session := request.Context().Value(ContextValues("session")).(*sessions.Session)
		githubUser, _ := session.Values["GithubUser"].(*User)
		providerName, _ := session.Values["Provider"].(string)

		login := ""
		if githubUser != nil {
			login = githubUser.Login
		}

		title := AppConfig.Title
		if title == "" {
			title = "gobookmarks"
		}
		if version == "dev" && !strings.HasPrefix(title, "dev: ") {
			title = "dev: " + title
		}

		ctx := context.WithValue(request.Context(), ContextValues("provider"), providerName)
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

type CoreData struct {
	Title       string
	AutoRefresh bool
	UserRef     string
	EditMode    bool
	Tab         int
}

type Configuration struct {
	GithubClientID       string   `json:"github_client_id"`
	GithubSecret         string   `json:"github_secret"`
	GitlabClientID       string   `json:"gitlab_client_id"`
	GitlabSecret         string   `json:"gitlab_secret"`
	ExternalURL          string   `json:"external_url"`
	OauthRedirectURL     string   `json:"-"`
	CssColumns           bool     `json:"css_columns"`
	DevMode              *bool    `json:"dev_mode"`
	Namespace            string   `json:"namespace"`
	Title                string   `json:"title"`
	GithubServer         string   `json:"github_server"`
	GitlabServer         string   `json:"gitlab_server"`
	FaviconCacheDir      string   `json:"favicon_cache_dir"`
	FaviconCacheSize     int64    `json:"favicon_cache_size"`
	LocalGitPath         string   `json:"local_git_path"`
	NoFooter             bool     `json:"no_footer"`
	SessionKey           string   `json:"session_key"`
	DBConnectionProvider string   `json:"db_connection_provider"`
	DBConnectionString   string   `json:"db_connection_string"`
	ProviderOrder        []string `json:"provider_order"`
	CommitsPerPage       int      `json:"commits_per_page"`

	data map[string]string
}

// TODO use for settings
func NewConfiguration() *Configuration {
	return &Configuration{
		data: make(map[string]string),
	}
}

func (c *Configuration) set(key, value string) {
	c.data[key] = value
}

func (c *Configuration) get(key string) string {
	return c.data[key]
}

func (c *Configuration) readConfiguration(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("File close error: %s", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		sep := strings.Index(line, "=")
		if sep >= 0 {
			key := line[:sep]
			value := line[sep+1:]
			c.set(key, value)
		}
	}
}

type ContextValues string
