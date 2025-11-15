package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/arran4/gobookmarks"
	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var (
	externalUrl string
	redirectUrl string
)

func NewServeCmd(version, commit, date string) *Command {
	var (
		cfgFlag            string
		ghIDFlag           string
		ghSecretFlag       string
		glIDFlag           string
		glSecretFlag       string
		urlFlag            string
		nsFlag             string
		titleFlag          string
		ghServerFlag       string
		glServerFlag       string
		faviconDirFlag     string
		faviconSizeFlag    string
		commitsPerPageFlag string
		localGitPathFlag   string
		dbProviderFlag     string
		dbConnFlag         string
		sessionKeyFlag     string
		providerOrderFlag  string
		columnFlag         bool
		noFooterFlag       bool
		devModeFlag        bool
		dumpConfig         bool
	)

	cmd := &Command{
		Name:  "serve",
		Short: "Starts the gobookmarks server",
		Long:  `Starts the gobookmarks server, serving the web UI and API.`,
		Run: func(cmd *Command, args []string) error {
			envPath := os.Getenv("GOBM_ENV_FILE")
			if envPath == "" {
				envPath = "/etc/gobookmarks/gobookmarks.env"
			}
			if err := LoadEnvFile(envPath); err != nil {
				log.Printf("unable to load env file %s: %v", envPath, err)
			}

			cfg := Config{
				GithubClientID:       os.Getenv("GITHUB_CLIENT_ID"),
				GithubSecret:         os.Getenv("GITHUB_SECRET"),
				GitlabClientID:       os.Getenv("GITLAB_CLIENT_ID"),
				GitlabSecret:         os.Getenv("GITLAB_SECRET"),
				ExternalURL:          os.Getenv("EXTERNAL_URL"),
				CssColumns:           os.Getenv("GBM_CSS_COLUMNS") != "",
				DevMode: func() *bool {
					if v, ok := os.LookupEnv("GBM_DEV_MODE"); ok {
						b, _ := strconv.ParseBool(v)
						return BP(b)
					}
					return nil
				}(),
				Namespace:            os.Getenv("GBM_NAMESPACE"),
				Title:                os.Getenv("GBM_TITLE"),
				GithubServer:         os.Getenv("GITHUB_SERVER"),
				GitlabServer:         os.Getenv("GITLAB_SERVER"),
				FaviconCacheDir:      os.Getenv("FAVICON_CACHE_DIR"),
				FaviconCacheSize: func() int64 {
					if v := os.Getenv("FAVICON_CACHE_SIZE"); v != "" {
						if i, err := strconv.ParseInt(v, 10, 64); err == nil {
							return i
						}
					}
					return 0
				}(),
				LocalGitPath:         os.Getenv("LOCAL_GIT_PATH"),
				DBConnectionProvider: os.Getenv("DB_CONNECTION_PROVIDER"),
				DBConnectionString:   os.Getenv("DB_CONNECTION_STRING"),
				NoFooter:             os.Getenv("GBM_NO_FOOTER") != "",
				SessionKey:           os.Getenv("SESSION_KEY"),
				ProviderOrder:        splitList(os.Getenv("PROVIDER_ORDER")),
				CommitsPerPage: func() int {
					if v := os.Getenv("COMMITS_PER_PAGE"); v != "" {
						if i, err := strconv.Atoi(v); err == nil {
							return i
						}
					}
					return 0
				}(),
			}

			configPath := DefaultConfigPath()

			if cmd.FlagSet.Lookup("config").Value.String() != "" {
				configPath = cfgFlag
			}

			cfgSpecified := cmd.FlagSet.Lookup("config").Value.String() != "" || os.Getenv("GOBM_CONFIG_FILE") != ""

			if fileCfg, found, err := LoadConfigFile(configPath); err == nil {
				if found {
					MergeConfig(&cfg, fileCfg)
				}
			} else {
				if os.IsNotExist(err) && !cfgSpecified {
					log.Printf("config file %s not found", configPath)
				} else {
					log.Printf("unable to load config file %s: %v", configPath, err)
					os.Exit(1)
				}
			}

			if cmd.FlagSet.Lookup("github-client-id").Value.String() != "" {
				cfg.GithubClientID = ghIDFlag
			}
			if cmd.FlagSet.Lookup("github-secret").Value.String() != "" {
				cfg.GithubSecret = ghSecretFlag
			}
			if cmd.FlagSet.Lookup("gitlab-client-id").Value.String() != "" {
				cfg.GitlabClientID = glIDFlag
			}
			if cmd.FlagSet.Lookup("gitlab-secret").Value.String() != "" {
				cfg.GitlabSecret = glSecretFlag
			}
			if cmd.FlagSet.Lookup("external-url").Value.String() != "" {
				cfg.ExternalURL = urlFlag
			}
			if cmd.FlagSet.Lookup("namespace").Value.String() != "" {
				cfg.Namespace = nsFlag
			}
			if cmd.FlagSet.Lookup("title").Value.String() != "" {
				cfg.Title = titleFlag
			}
			if cmd.FlagSet.Lookup("css-columns").Value.String() == "true" {
				cfg.CssColumns = columnFlag
			}
			if cmd.FlagSet.Lookup("no-footer").Value.String() == "true" {
				cfg.NoFooter = noFooterFlag
			}
			if cmd.FlagSet.Lookup("dev-mode").Value.String() == "true" {
				cfg.DevMode = BP(devModeFlag)
			}
			if cmd.FlagSet.Lookup("github-server").Value.String() != "" {
				cfg.GithubServer = ghServerFlag
			}
			if cmd.FlagSet.Lookup("favicon-cache-dir").Value.String() != "" {
				cfg.FaviconCacheDir = faviconDirFlag
			}
			if cmd.FlagSet.Lookup("favicon-cache-size").Value.String() != "" {
				if i, err := strconv.ParseInt(faviconSizeFlag, 10, 64); err == nil {
					cfg.FaviconCacheSize = i
				}
			}
			if cmd.FlagSet.Lookup("commits-per-page").Value.String() != "" {
				if i, err := strconv.Atoi(commitsPerPageFlag); err == nil {
					cfg.CommitsPerPage = i
				}
			}
			if cmd.FlagSet.Lookup("gitlab-server").Value.String() != "" {
				cfg.GitlabServer = glServerFlag
			}
			if cmd.FlagSet.Lookup("local-git-path").Value.String() != "" {
				cfg.LocalGitPath = localGitPathFlag
			}
			if cmd.FlagSet.Lookup("db-provider").Value.String() != "" {
				cfg.DBConnectionProvider = dbProviderFlag
			}
			if cmd.FlagSet.Lookup("db-conn").Value.String() != "" {
				cfg.DBConnectionString = dbConnFlag
			}
			if cmd.FlagSet.Lookup("session-key").Value.String() != "" {
				cfg.SessionKey = sessionKeyFlag
			}
			if cmd.FlagSet.Lookup("provider-order").Value.String() != "" {
				cfg.ProviderOrder = splitList(providerOrderFlag)
			}

			if dumpConfig {
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
			if devModeFlag {
				DevMode = devModeFlag
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
			externalUrl = strings.TrimRight(cfg.ExternalURL, "/")
			redirectUrl = JoinURL(externalUrl, "oauth2Callback")
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

			r := mux.NewRouter()

			r.Use(UserAdderMiddleware)
			r.Use(CoreAdderMiddleware)

			r.HandleFunc("/main.css", func(writer http.ResponseWriter, request *http.Request) {
				_, _ = writer.Write(GetMainCSSData())
			}).Methods("GET")
			r.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, request *http.Request) {
				_, _ = writer.Write(GetFavicon())
			}).Methods("GET")

			// Development helpers to toggle layout mode
			if DevMode {
				r.HandleFunc("/_css", runHandlerChain(EnableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
				r.HandleFunc("/_table", runHandlerChain(DisableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
			}

			// News
			r.Handle("/", http.HandlerFunc(runTemplate("mainPage.gohtml"))).Methods("GET")
			r.HandleFunc("/", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

			r.HandleFunc("/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(HasError())
			r.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
			r.HandleFunc("/edit", runHandlerChain(BookmarksEditCreateAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Create"))
			r.HandleFunc("/edit", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

			r.HandleFunc("/startEditMode", runHandlerChain(StartEditMode, redirectToHandlerTabPage("/"))).Methods("POST", "GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/stopEditMode", runHandlerChain(StopEditMode, redirectToHandlerTabPage("/"))).Methods("POST", "GET").MatcherFunc(RequiresAnAccount())

			r.HandleFunc("/editCategory", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/editCategory", runHandlerChain(EditCategoryPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/editCategory", runHandlerChain(CategoryEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
			r.HandleFunc("/editCategory", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")
			r.HandleFunc("/addCategory", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/addCategory", runHandlerChain(AddCategoryPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/addCategory", runHandlerChain(CategoryAddSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
			r.HandleFunc("/addCategory", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")
			r.HandleFunc("/moveCategory", runHandlerChain(CategoryMoveBeforeAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/moveCategoryEnd", runHandlerChain(CategoryMoveEndAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/moveCategoryNewColumn", runHandlerChain(CategoryMoveNewColumnAction)).Methods("POST").MatcherFunc(RequiresAnAccount())

			r.HandleFunc("/editTab", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/editTab", runHandlerChain(EditTabPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/editTab", runHandlerChain(TabEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
			r.HandleFunc("/editTab", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

			r.HandleFunc("/editPage", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/editPage", runHandlerChain(EditPagePage)).Methods("GET").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/editPage", runHandlerChain(PageEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
			r.HandleFunc("/editPage", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

			r.HandleFunc("/moveTab", runHandlerChain(MoveTabAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/movePage", runHandlerChain(MovePageAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
			r.HandleFunc("/moveEntry", runHandlerChain(MoveEntryAction)).Methods("POST").MatcherFunc(RequiresAnAccount())

			r.HandleFunc("/history", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/history", runTemplate("history.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

			r.HandleFunc("/history/commits", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
			r.HandleFunc("/status", runTemplate("statusPage.gohtml")).Methods("GET")
			r.HandleFunc("/history/commits", runTemplate("historyCommits.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

			r.HandleFunc("/login", runTemplate("loginPage.gohtml")).Methods("GET")
			r.HandleFunc("/login/git", runTemplate("gitLoginPage.gohtml")).Methods("GET")
			r.HandleFunc("/login/git", runHandlerChain(GitLoginAction, redirectToHandler("/"))).Methods("POST")
			r.HandleFunc("/signup/git", runHandlerChain(GitSignupAction, redirectToHandler("/login/git"))).Methods("POST")
			r.HandleFunc("/login/sql", runTemplate("sqlLoginPage.gohtml")).Methods("GET")
			r.HandleFunc("/login/sql", runHandlerChain(SqlLoginAction, redirectToHandler("/"))).Methods("POST")
			r.HandleFunc("/signup/sql", runHandlerChain(SqlSignupAction, redirectToHandler("/login/sql"))).Methods("POST")
			r.HandleFunc("/login/{provider}", runHandlerChain(LoginWithProvider)).Methods("GET")
			r.HandleFunc("/logout", runHandlerChain(UserLogoutAction, runTemplate("logoutPage.gohtml"))).Methods("GET")
			r.HandleFunc("/oauth2Callback", runHandlerChain(Oauth2CallbackPage, redirectToHandler("/"))).Methods("GET")

			r.HandleFunc("/proxy/favicon", FaviconProxyHandler).Methods("GET")

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
		},
	}
	cmd.FlagSet.StringVar(&cfgFlag, "config", "", "path to config file")
	cmd.FlagSet.StringVar(&ghIDFlag, "github-client-id", "", "GitHub OAuth client ID")
	cmd.FlagSet.StringVar(&ghSecretFlag, "github-secret", "", "GitHub OAuth client secret")
	cmd.FlagSet.StringVar(&glIDFlag, "gitlab-client-id", "", "GitLab OAuth client ID")
	cmd.FlagSet.StringVar(&glSecretFlag, "gitlab-secret", "", "GitLab OAuth client secret")
	cmd.FlagSet.StringVar(&urlFlag, "external-url", "", "external URL")
	cmd.FlagSet.StringVar(&nsFlag, "namespace", "", "repository namespace")
	cmd.FlagSet.StringVar(&titleFlag, "title", "", "site title")
	cmd.FlagSet.StringVar(&faviconDirFlag, "favicon-cache-dir", "", "directory for cached favicons")
	cmd.FlagSet.StringVar(&faviconSizeFlag, "favicon-cache-size", "", "max size of favicon cache in bytes")
	cmd.FlagSet.StringVar(&commitsPerPageFlag, "commits-per-page", "", "commits per page")
	cmd.FlagSet.StringVar(&ghServerFlag, "github-server", "", "GitHub base URL")
	cmd.FlagSet.StringVar(&glServerFlag, "gitlab-server", "", "GitLab base URL")
	cmd.FlagSet.StringVar(&localGitPathFlag, "local-git-path", "", "directory for local git provider")
	cmd.FlagSet.StringVar(&dbProviderFlag, "db-provider", "", "SQL driver name")
	cmd.FlagSet.StringVar(&dbConnFlag, "db-conn", "", "SQL connection string")
	cmd.FlagSet.StringVar(&sessionKeyFlag, "session-key", "", "session cookie key")
	cmd.FlagSet.StringVar(&providerOrderFlag, "provider-order", "", "comma-separated provider order")
	cmd.FlagSet.BoolVar(&columnFlag, "css-columns", false, "use CSS columns")
	cmd.FlagSet.BoolVar(&noFooterFlag, "no-footer", false, "disable footer on pages")
	cmd.FlagSet.BoolVar(&devModeFlag, "dev-mode", false, "enable dev mode helpers")
	cmd.FlagSet.BoolVar(&dumpConfig, "dump-config", false, "print merged config and exit")
	return cmd
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

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func CreatePEMFiles() {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Your Organization"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certFile, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("Failed to create cert.pem file: %v", err)
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}

	keyFile, err := os.Create("key.pem")
	if err != nil {
		log.Fatalf("Failed to create key.pem file: %v", err)
	}
	defer keyFile.Close()
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		log.Fatalf("Failed to marshal private key: %v", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
}

func runHandlerChain(chain ...any) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, each := range chain {
			switch each := each.(type) {
			case http.Handler:
				each.ServeHTTP(w, r)
			case http.HandlerFunc:
				each(w, r)
			case func(http.ResponseWriter, *http.Request):
				each(w, r)
			case func(http.ResponseWriter, *http.Request) error:
				if err := each(w, r); err != nil {
					if errors.Is(err, ErrHandled) {
						return
					}
					if errors.Is(err, ErrSignedOut) {
						if logoutErr := UserLogoutAction(w, r); logoutErr != nil {
							log.Printf("logout error: %v", logoutErr)
						}
						type Data struct{ *CoreData }
						if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "logoutPage.gohtml", Data{r.Context().Value(ContextValues("coreData")).(*CoreData)}); err != nil {
							log.Printf("Logout Template Error: %s", err)
							http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						}
						return
					}

					var uerr UserError
					if errors.As(err, &uerr) {
						dest := r.Referer()
						if dest == "" {
							dest = r.URL.Path
							if q := r.URL.Query(); len(q) > 0 {
								dest += "?" + q.Encode()
							}
						}
						u, parseErr := url.Parse(dest)
						if parseErr != nil {
							log.Printf("user error parse referer: %v", parseErr)
						} else {
							q := u.Query()
							q.Set("error", uerr.Msg)
							u.RawQuery = q.Encode()
							http.Redirect(w, r, u.String(), http.StatusSeeOther)
							return
						}
					}

					var serr SystemError
					display := "Internal error"
					if errors.As(err, &serr) {
						display = serr.Msg
						err = serr.Err
					}

					log.Printf("handler error: %v", err)

					type ErrorData struct {
						*CoreData
						Error string
					}
					if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
						CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
						Error:    display,
					}); err != nil {
						log.Printf("Error Template Error: %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
			default:
				log.Panicf("unknown input: %s", reflect.TypeOf(each))
			}
		}
	}
}

func runTemplate(tmpl string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type Data struct {
			*CoreData
			Error string
		}

		data := Data{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    r.URL.Query().Get("error"),
		}

		var buf bytes.Buffer
		err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(&buf, tmpl, data)
		if err == nil {
			_, _ = io.Copy(w, &buf)
			return
		}

		if errors.Is(err, ErrSignedOut) {
			if logoutErr := UserLogoutAction(w, r); logoutErr != nil {
				log.Printf("logout error: %v", logoutErr)
			}
			type LogoutData struct{ *CoreData }
			if tplErr := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "logoutPage.gohtml", LogoutData{data.CoreData}); tplErr != nil {
				log.Printf("Logout Template Error: %v", tplErr)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		var serr SystemError
		display := "Internal error"
		if errors.As(err, &serr) {
			display = serr.Msg
			err = serr.Err
		}

		log.Printf("Template %s error: %v", tmpl, err)

		type ErrorData struct {
			*CoreData
			Error string
		}

		if tplErr := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: data.CoreData,
			Error:    display,
		}); tplErr != nil {
			log.Printf("Error Template Error: %v", tplErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

func redirectToHandler(toUrl string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, toUrl, http.StatusTemporaryRedirect)
	})
}

func redirectToHandlerBranchToRef(toUrl string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse(toUrl)
		qs := u.Query()
		qs.Set("ref", "refs/heads/"+r.PostFormValue("branch"))
		tab := r.PostFormValue("tab")
		if v, ok := r.Context().Value(ContextValues("redirectTab")).(string); ok {
			tab = v
		}
		if tab != "" {
			qs.Set("tab", tab)
		}
		page := r.PostFormValue("page")
		if v, ok := r.Context().Value(ContextValues("redirectPage")).(string); ok {
			page = v
		}
		if page != "" {
			u.Fragment = "page" + page
		}
		if edit := r.URL.Query().Get("edit"); edit != "" {
			qs.Set("edit", edit)
		}
		u.RawQuery = qs.Encode()
		http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
	})
}

func redirectToHandlerTabPage(toUrl string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse(toUrl)
		qs := u.Query()
		if tab := r.URL.Query().Get("tab"); tab != "" {
			qs.Set("tab", tab)
		}
		if page := r.URL.Query().Get("page"); page != "" {
			u.Fragment = "page" + page
		}
		if edit := r.URL.Query().Get("edit"); edit != "" {
			qs.Set("edit", edit)
		}
		u.RawQuery = qs.Encode()
		http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
	})
}

func RequiresAnAccount() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		var session *sessions.Session
		sessioni := request.Context().Value(ContextValues("session"))
		if sessioni == nil {
			var err error
			session, err = SessionStore.Get(request, SessionName)
			if err != nil {
				return false
			}
		} else {
			var ok bool
			session, ok = sessioni.(*sessions.Session)
			if !ok {
				return false
			}
		}
		if v, ok := session.Values["version"].(string); !ok || v != version {
			return false
		}
		githubUser, ok := session.Values["GithubUser"].(*User)
		return ok && githubUser != nil
	}
}

func TaskMatcher(taskName string) mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.PostFormValue("task") == taskName
	}
}

func ModeMatcher(modeName string) mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.URL.Query().Get("mode") == modeName
	}
}

func HasError() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.URL.Query().Has("error")
	}
}

func NoTask() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.PostFormValue("task") == ""
	}
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
