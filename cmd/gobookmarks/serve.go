package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/arran4/gobookmarks"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type ServeCommand struct {
	parent Command
	Flags  *flag.FlagSet

	GithubClientID       stringFlag
	GithubSecret         stringFlag
	GitlabClientID       stringFlag
	GitlabSecret         stringFlag
	ExternalURL          stringFlag
	Namespace            stringFlag
	Title                stringFlag
	FaviconCacheDir      stringFlag
	FaviconCacheSize     stringFlag
	FaviconMaxCacheCount stringFlag
	CommitsPerPage       stringFlag
	GithubServer         stringFlag
	GitlabServer         stringFlag
	LocalGitPath         stringFlag
	DbProvider           stringFlag
	DbConn               stringFlag
	SessionKey           stringFlag
	ProviderOrder        stringFlag
	CssColumns           boolFlag
	NoFooter             boolFlag
	DevMode              boolFlag
	DumpConfig           boolFlag
}

func (rc *RootCommand) NewServeCommand() (*ServeCommand, error) {
	c := &ServeCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("serve", flag.ContinueOnError),
	}

	c.Flags.Var(&c.GithubClientID, "github-client-id", "GitHub OAuth client ID")
	c.Flags.Var(&c.GithubSecret, "github-secret", "GitHub OAuth client secret")
	c.Flags.Var(&c.GitlabClientID, "gitlab-client-id", "GitLab OAuth client ID")
	c.Flags.Var(&c.GitlabSecret, "gitlab-secret", "GitLab OAuth client secret")
	c.Flags.Var(&c.ExternalURL, "external-url", "external URL")
	c.Flags.Var(&c.Namespace, "namespace", "repository namespace")
	c.Flags.Var(&c.Title, "title", "site title")
	c.Flags.Var(&c.FaviconCacheDir, "favicon-cache-dir", "directory for cached favicons")
	c.Flags.Var(&c.FaviconCacheSize, "favicon-cache-size", "max size of favicon cache in bytes")
	c.Flags.Var(&c.FaviconMaxCacheCount, "favicon-max-cache-count", "max number of items in favicon cache")
	c.Flags.Var(&c.CommitsPerPage, "commits-per-page", "commits per page")
	c.Flags.Var(&c.GithubServer, "github-server", "GitHub base URL")
	c.Flags.Var(&c.GitlabServer, "gitlab-server", "GitLab base URL")
	c.Flags.Var(&c.LocalGitPath, "local-git-path", "directory for local git provider")
	c.Flags.Var(&c.DbProvider, "db-provider", "SQL driver name")
	c.Flags.Var(&c.DbConn, "db-conn", "SQL connection string")
	c.Flags.Var(&c.SessionKey, "session-key", "session cookie key")
	c.Flags.Var(&c.ProviderOrder, "provider-order", "comma-separated provider order")
	c.Flags.Var(&c.CssColumns, "css-columns", "use CSS columns")
	c.Flags.Var(&c.NoFooter, "no-footer", "disable footer on pages")
	c.Flags.Var(&c.DevMode, "dev-mode", "enable dev mode helpers")
	c.Flags.Var(&c.DumpConfig, "dump-config", "print merged config and exit")

	return c, nil
}

func (c *ServeCommand) Name() string {
	return c.Flags.Name()
}

func (c *ServeCommand) Parent() Command {
	return c.parent
}

func (c *ServeCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ServeCommand) Subcommands() []Command {
	return nil
}

func (c *ServeCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}
	cfg := c.parent.(*RootCommand).cfg

	if c.GithubClientID.set {
		cfg.GithubClientID = c.GithubClientID.value
	}
	if c.GithubSecret.set {
		cfg.GithubSecret = c.GithubSecret.value
	}
	if c.GitlabClientID.set {
		cfg.GitlabClientID = c.GitlabClientID.value
	}
	if c.GitlabSecret.set {
		cfg.GitlabSecret = c.GitlabSecret.value
	}
	if c.ExternalURL.set {
		cfg.ExternalURL = c.ExternalURL.value
	}
	if c.Namespace.set {
		cfg.Namespace = c.Namespace.value
	}
	if c.Title.set {
		cfg.Title = c.Title.value
	}
	if c.CssColumns.set {
		cfg.CssColumns = c.CssColumns.value
	}
	if c.NoFooter.set {
		cfg.NoFooter = c.NoFooter.value
	}
	if c.DevMode.set {
		cfg.DevMode = BP(c.DevMode.value)
	}
	if c.GithubServer.set {
		cfg.GithubServer = c.GithubServer.value
	}
	if c.FaviconCacheDir.set {
		cfg.FaviconCacheDir = c.FaviconCacheDir.value
	}
	if c.FaviconCacheSize.set {
		if i, err := strconv.ParseInt(c.FaviconCacheSize.value, 10, 64); err == nil {
			cfg.FaviconCacheSize = i
		}
	}
	if c.FaviconMaxCacheCount.set {
		if i, err := strconv.Atoi(c.FaviconMaxCacheCount.value); err == nil {
			cfg.FaviconMaxCacheCount = i
		}
	}
	if c.CommitsPerPage.set {
		if i, err := strconv.Atoi(c.CommitsPerPage.value); err == nil {
			cfg.CommitsPerPage = i
		}
	}
	if c.GitlabServer.set {
		cfg.GitlabServer = c.GitlabServer.value
	}
	if c.LocalGitPath.set {
		cfg.LocalGitPath = c.LocalGitPath.value
	}
	if c.DbProvider.set {
		cfg.DBConnectionProvider = c.DbProvider.value
	}
	if c.DbConn.set {
		cfg.DBConnectionString = c.DbConn.value
	}
	if c.SessionKey.set {
		cfg.SessionKey = c.SessionKey.value
	}
	if c.ProviderOrder.set {
		cfg.ProviderOrder = splitList(c.ProviderOrder.value)
	}

	if c.DumpConfig.value {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	UseCssColumns = cfg.CssColumns
	Namespace = cfg.Namespace
	RepoName = GetBookmarksRepoName()
	SiteTitle = cfg.Title
	NoFooter = cfg.NoFooter
	DevMode = version == "dev"
	if cfg.DevMode != nil {
		DevMode = *cfg.DevMode
	}

	if cfg.GithubServer != "" {
		GithubServer = cfg.GithubServer
	}
	if cfg.GitlabServer != "" {
		GitlabServer = cfg.GitlabServer
	}
	if cfg.FaviconCacheDir != "" {
		FaviconCacheDir = cfg.FaviconCacheDir
	}
	if cfg.FaviconCacheSize != 0 {
		FaviconCacheSize = cfg.FaviconCacheSize
	} else {
		FaviconCacheSize = DefaultFaviconCacheSize
	}
	if cfg.FaviconMaxCacheCount != 0 {
		FaviconMaxCacheCount = cfg.FaviconMaxCacheCount
	} else {
		FaviconMaxCacheCount = DefaultFaviconMaxCacheCount
	}
	if cfg.CommitsPerPage != 0 {
		CommitsPerPage = cfg.CommitsPerPage
	} else {
		CommitsPerPage = DefaultCommitsPerPage
	}
	if cfg.LocalGitPath != "" {
		LocalGitPath = cfg.LocalGitPath
	}
	if cfg.DBConnectionProvider != "" {
		DBConnectionProvider = cfg.DBConnectionProvider
	}
	if cfg.DBConnectionString != "" {
		DBConnectionString = cfg.DBConnectionString
	}
	githubID := cfg.GithubClientID
	githubSecret := cfg.GithubSecret
	gitlabID := cfg.GitlabClientID
	gitlabSecret := cfg.GitlabSecret
	externalUrl := strings.TrimRight(cfg.ExternalURL, "/")
	redirectUrl := JoinURL(externalUrl, "oauth2Callback")
	GithubClientID = githubID
	GithubClientSecret = githubSecret
	GitlabClientID = gitlabID
	GitlabClientSecret = gitlabSecret
	OauthRedirectURL = redirectUrl

	SetProviderOrder(cfg.ProviderOrder)

	SessionName = "gobookmarks"
	SessionStore = sessions.NewCookieStore(loadSessionKey(cfg))
	if len(ProviderNames()) == 0 {
		return errors.New("no providers compiled")
	}
	if len(ConfiguredProviderNames()) == 0 {
		return errors.New("no providers available")
	}

	// Create RouterConfig
	routerCfg := &RouterConfig{
		SessionStore: SessionStore, // Globals should be initialized by now (lines 248-257)
		SessionName:  SessionName,
		ExternalURL:  cfg.ExternalURL,
		BaseURL:      "", // Root
		DevMode:      *cfg.DevMode,
	}

	r := NewRouter(routerCfg)

	http.Handle("/", r)

	if !fileExists("cert.pem") || !fileExists("key.pem") {
		CreatePEMFiles()
	}

	log.Printf("gobookmarks: %s, commit %s, built at %s", version, commit, date)
	SetVersion(version, commit, date)
	RepoName = GetBookmarksRepoName()
	log.Printf("Redirect URL configured to: %s", redirectUrl)
	log.Println("Server started on http://localhost:8080")
	log.Println("Server started on https://localhost:8443")

	// Create a context with a cancel function
	_, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation when main exits

	// Create an HTTP server with a handler
	httpServer := &http.Server{
		Addr: ":8080",
	}

	// Create an HTTPS server with a handler
	httpsServer := &http.Server{
		Addr: ":8443",
	}

	var sigCh chan os.Signal
	// Handle ^C signal (SIGINT) to gracefully shut down the servers
	go func() {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh

		fmt.Println("Shutting down gracefully...")

		// Cancel the context to signal shutdown to both servers
		cancel()

		// Give some time for active connections to finish
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server error during shutdown: %v", err)
		}

		if err := httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server error during shutdown: %v", err)
		}

		fmt.Println("Servers gracefully shut down.")
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the HTTP server
	go func() {
		defer wg.Done()
		fmt.Println("HTTP server listening on :8080...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start the HTTPS server (TLS/SSL)
	go func() {
		defer wg.Done()
		fmt.Println("HTTPS server listening on :8443...")
		if err := httpsServer.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	wg.Wait()
	return nil
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
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
