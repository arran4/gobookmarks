package gobookmarks

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

const (
	DefaultFaviconCacheSize     int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconMaxCacheCount int           = 1000
	DefaultFaviconCacheMaxAge   time.Duration = 24 * time.Hour
	DefaultCommitsPerPage       int           = 100
)

func init() {
	gob.Register(&User{})
	gob.Register(&oauth2.Token{})
}

func CoreAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cfg := request.Context().Value(ContextValues("configuration")).(*Configuration)
		session := request.Context().Value(ContextValues("session")).(*sessions.Session)
		githubUser, _ := session.Values["GithubUser"].(*User)
		providerName, _ := session.Values["Provider"].(string)

		login := ""
		if githubUser != nil {
			login = githubUser.Login
		}

		title := cfg.GetTitle()
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
			UserRef:      login,
			Title:        title,
			EditMode:     editMode,
			Tab:          tab,
			requestCache: &requestCache{data: make(map[string]*bookmarkCacheEntry)},
		})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

type CoreData struct {
	Title        string
	AutoRefresh  bool
	UserRef      string
	EditMode     bool
	Tab          int
	requestCache *requestCache
}

type Configuration struct {
	Config
	SessionStore sessions.Store
	SessionName  string
}

func NewConfiguration(config Config) *Configuration {
	c := &Configuration{
		Config: config,
	}
	c.SessionName = "gobookmarks"
	c.SessionStore = sessions.NewCookieStore(loadSessionKey(config))
	return c
}

func loadSessionKey(cfg Config) []byte {
	if cfg.SessionKey != "" {
		return []byte(cfg.SessionKey)
	}

	path := DefaultSessionKeyPath(false)
	if b, err := os.ReadFile(path); err == nil {
		return bytes.TrimSpace(b)
	}

	key := securecookie.GenerateRandomKey(32)
	if key == nil {
		log.Fatal("unable to generate session key")
	}

	path = DefaultSessionKeyPath(true)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if err := os.WriteFile(path, key, 0o600); err != nil {
			log.Printf("unable to write session key file %s: %v; sessions will not persist", path, err)
		}
	} else {
		log.Printf("unable to create session key directory %s: %v; sessions will not persist", filepath.Dir(path), err)
	}

	return key
}

func ConfigMiddleware(cfg *Configuration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ContextValues("configuration"), cfg)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (c *Configuration) GetRepoName() string {
	ns := c.Namespace
	if strings.EqualFold(version, "dev") {
		if ns == "" {
			ns = version
		}
	}

	name := "MyBookmarks"
	if ns != "" {
		name += "-" + ns
	}
	return name
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

func (c *Configuration) GetTitle() string {
	return c.Title
}

func (c *Configuration) GetUseCssColumns() bool {
	return c.CssColumns
}

func (c *Configuration) GetDevMode() bool {
	if c.DevMode != nil {
		return *c.DevMode
	}
	return version == "dev"
}

func (c *Configuration) GetNoFooter() bool {
	return c.NoFooter
}

func (c *Configuration) GetCommitsPerPage() int {
	if c.CommitsPerPage != 0 {
		return c.CommitsPerPage
	}
	return DefaultCommitsPerPage
}

func (c *Configuration) GetFaviconCacheDir() string {
	return c.FaviconCacheDir
}

func (c *Configuration) GetFaviconCacheSize() int64 {
	if c.FaviconCacheSize != 0 {
		return c.FaviconCacheSize
	}
	return DefaultFaviconCacheSize
}

func (c *Configuration) GetFaviconMaxCacheCount() int {
	if c.FaviconMaxCacheCount != 0 {
		return c.FaviconMaxCacheCount
	}
	return DefaultFaviconMaxCacheCount
}

func (c *Configuration) GetProviderOrder() []string {
	return c.ProviderOrder
}

func (c *Configuration) GetDBConnectionProvider() string {
	return c.DBConnectionProvider
}

func (c *Configuration) GetDBConnectionString() string {
	return c.DBConnectionString
}

func (c *Configuration) GetLocalGitPath() string {
	return c.LocalGitPath
}

func (c *Configuration) GetRedirectURL() string {
	externalUrl := strings.TrimRight(c.ExternalURL, "/")
	return JoinURL(externalUrl, "oauth2Callback")
}

func (c *Configuration) GetProviderNames() []string {
	defaultOrder := GetDefaultProviderOrder()
	if len(c.ProviderOrder) == 0 {
		return defaultOrder
	}

	known := make(map[string]bool)
	for _, n := range defaultOrder {
		known[n] = true
	}

	var final []string
	seen := make(map[string]bool)
	for _, n := range c.ProviderOrder {
		if known[n] && !seen[n] {
			final = append(final, n)
			seen[n] = true
		}
	}
	var rest []string
	for _, n := range defaultOrder {
		if !seen[n] {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	return append(final, rest...)
}

func (c *Configuration) GetConfiguredProviderNames() []string {
	var names []string
	for _, n := range c.GetProviderNames() {
		if c.GetProviderCreds(n) != nil {
			names = append(names, n)
		}
	}
	return names
}

type ContextValues string
