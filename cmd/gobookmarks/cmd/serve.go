package cmd

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
	. "github.com/arran4/gobookmarks"
	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/spf13/cobra"
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
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the gobookmarks server",
	Long:  `Starts the gobookmarks server, serving the web UI and API.`,
	Run:   runServe,
}

type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) Set(v string) error { s.value = v; s.set = true; return nil }
func (s *stringFlag) String() string     { return s.value }
func (s *stringFlag) Type() string       { return "string" }

type boolFlag struct {
	value bool
	set   bool
}

func (b *boolFlag) Set(v string) error {
	val, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.value = val
	b.set = true
	return nil
}
func (b *boolFlag) String() string { return strconv.FormatBool(b.value) }
func (b *boolFlag) Type() string   { return "bool" }

var (
	cfgFlag            stringFlag
	ghIDFlag           stringFlag
	ghSecretFlag       stringFlag
	glIDFlag           stringFlag
	glSecretFlag       stringFlag
	urlFlag            stringFlag
	nsFlag             stringFlag
	titleFlag          stringFlag
	ghServerFlag       stringFlag
	glServerFlag       stringFlag
	faviconDirFlag     stringFlag
	faviconSizeFlag    stringFlag
	commitsPerPageFlag stringFlag
	localGitPathFlag   stringFlag
	dbProviderFlag     stringFlag
	dbConnFlag         stringFlag
	sessionKeyFlag     stringFlag
	providerOrderFlag  stringFlag
	columnFlag         boolFlag
	noFooterFlag       boolFlag
	devModeFlag        boolFlag
	dumpConfig         bool
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().Var(&cfgFlag, "config", "path to config file")
	serveCmd.Flags().Var(&ghIDFlag, "github-client-id", "GitHub OAuth client ID")
	serveCmd.Flags().Var(&ghSecretFlag, "github-secret", "GitHub OAuth client secret")
	serveCmd.Flags().Var(&glIDFlag, "gitlab-client-id", "GitLab OAuth client ID")
	serveCmd.Flags().Var(&glSecretFlag, "gitlab-secret", "GitLab OAuth client secret")
	serveCmd.Flags().Var(&urlFlag, "external-url", "external URL")
	serveCmd.Flags().Var(&nsFlag, "namespace", "repository namespace")
	serveCmd.Flags().Var(&titleFlag, "title", "site title")
	serveCmd.Flags().Var(&faviconDirFlag, "favicon-cache-dir", "directory for cached favicons")
	serveCmd.Flags().Var(&faviconSizeFlag, "favicon-cache-size", "max size of favicon cache in bytes")
	serveCmd.Flags().Var(&commitsPerPageFlag, "commits-per-page", "commits per page")
	serveCmd.Flags().Var(&ghServerFlag, "github-server", "GitHub base URL")
	serveCmd.Flags().Var(&glServerFlag, "gitlab-server", "GitLab base URL")
	serveCmd.Flags().Var(&localGitPathFlag, "local-git-path", "directory for local git provider")
	serveCmd.Flags().Var(&dbProviderFlag, "db-provider", "SQL driver name")
	serveCmd.Flags().Var(&dbConnFlag, "db-conn", "SQL connection string")
	serveCmd.Flags().Var(&sessionKeyFlag, "session-key", "session cookie key")
	serveCmd.Flags().Var(&providerOrderFlag, "provider-order", "comma-separated provider order")
	serveCmd.Flags().Var(&columnFlag, "css-columns", "use CSS columns")
	serveCmd.Flags().Var(&noFooterFlag, "no-footer", "disable footer on pages")
	serveCmd.Flags().Var(&devModeFlag, "dev-mode", "enable dev mode helpers")
	serveCmd.Flags().BoolVar(&dumpConfig, "dump-config", false, "print merged config and exit")
}

func runServe(cmd *cobra.Command, args []string) {
	envPath := os.Getenv("GOBM_ENV_FILE")
	if envPath == "" {
		envPath = "/etc/gobookmarks/gobookmarks.env"
	}
	if err := LoadEnvFile(envPath); err != nil {
		log.Printf("unable to load env file %s: %v", envPath, err)
	}

	cfg := Config{
		GithubClientID: os.Getenv("GITHUB_CLIENT_ID"),
		GithubSecret:   os.Getenv("GITHUB_SECRET"),
		GitlabClientID: os.Getenv("GITLAB_CLIENT_ID"),
		GitlabSecret:   os.Getenv("GITLAB_SECRET"),
		ExternalURL:    os.Getenv("EXTERNAL_URL"),
		CssColumns:     os.Getenv("GBM_CSS_COLUMNS") != "",
		DevMode: func() *bool {
			if v, ok := os.LookupEnv("GBM_DEV_MODE"); ok {
				b, _ := strconv.ParseBool(v)
				return BP(b)
			}
			return nil
		}(),
		Namespace:       os.Getenv("GBM_NAMESPACE"),
		Title:           os.Getenv("GBM_TITLE"),
		GithubServer:    os.Getenv("GITHUB_SERVER"),
		GitlabServer:    os.Getenv("GITLAB_SERVER"),
		FaviconCacheDir: os.Getenv("FAVICON_CACHE_DIR"),
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

	if cfgFlag.set {
		configPath = cfgFlag.value
	}

	cfgSpecified := cfgFlag.set || os.Getenv("GOBM_CONFIG_FILE") != ""

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

	if ghIDFlag.set {
		cfg.GithubClientID = ghIDFlag.value
	}
	if ghSecretFlag.set {
		cfg.GithubSecret = ghSecretFlag.value
	}
	if glIDFlag.set {
		cfg.GitlabClientID = glIDFlag.value
	}
	if glSecretFlag.set {
		cfg.GitlabSecret = glSecretFlag.value
	}
	if urlFlag.set {
		cfg.ExternalURL = urlFlag.value
	}
	if nsFlag.set {
		cfg.Namespace = nsFlag.value
	}
	if titleFlag.set {
		cfg.Title = titleFlag.value
	}
	if columnFlag.set {
		cfg.CssColumns = columnFlag.value
	}
	if noFooterFlag.set {
		cfg.NoFooter = noFooterFlag.value
	}
	if devModeFlag.set {
		cfg.DevMode = BP(devModeFlag.value)
	}
	if ghServerFlag.set {
		cfg.GithubServer = ghServerFlag.value
	}
	if faviconDirFlag.set {
		cfg.FaviconCacheDir = faviconDirFlag.value
	}
	if faviconSizeFlag.set {
		if i, err := strconv.ParseInt(faviconSizeFlag.value, 10, 64); err == nil {
			cfg.FaviconCacheSize = i
		}
	}
	if commitsPerPageFlag.set {
		if i, err := strconv.Atoi(commitsPerPageFlag.value); err == nil {
			cfg.CommitsPerPage = i
		}
	}
	if glServerFlag.set {
		cfg.GitlabServer = glServerFlag.value
	}
	if localGitPathFlag.set {
		cfg.LocalGitPath = localGitPathFlag.value
	}
	if dbProviderFlag.set {
		cfg.DBConnectionProvider = dbProviderFlag.value
	}
	if dbConnFlag.set {
		cfg.DBConnectionString = dbConnFlag.value
	}
	if sessionKeyFlag.set {
		cfg.SessionKey = sessionKeyFlag.value
	}
	if providerOrderFlag.set {
		cfg.ProviderOrder = splitList(providerOrderFlag.value)
	}

	if dumpConfig {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		return
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
	if devModeFlag.set {
		DevMode = devModeFlag.value
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
		fmt.Println("no providers compiled")
		os.Exit(-1)
	}
	if len(ConfiguredProviderNames()) == 0 {
		fmt.Println("no providers available")
		os.Exit(-1)
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

	if DevMode {
		r.HandleFunc("/_css", runHandlerChain(EnableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
		r.HandleFunc("/_table", runHandlerChain(DisableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
	}

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

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer := &http.Server{
		Addr: ":8080",
	}

	httpsServer := &http.Server{
		Addr: ":8443",
	}

	var sigCh chan os.Signal
	go func() {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh

		fmt.Println("Shutting down gracefully...")

		cancel()

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
	go func() {
		defer wg.Done()
		fmt.Println("HTTP server listening on :8080...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		fmt.Println("HTTPS server listening on :8443...")
		if err := httpsServer.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	wg.Wait()
}

func CreatePEMFiles() {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

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

var (
	externalUrl string
	redirectUrl string
)
