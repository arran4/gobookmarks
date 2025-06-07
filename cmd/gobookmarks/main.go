package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	. "github.com/arran4/gobookmarks"
	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	clientID     string
	clientSecret string
	externalUrl  string
	redirectUrl  string
	version      = "dev"
	commit       = "none"
	date         = "unknown"
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func main() {

	envPath := os.Getenv("GOBM_ENV_FILE")
	if envPath == "" {
		envPath = "/etc/gobookmarks/gobookmarks.env"
	}
	if err := LoadEnvFile(envPath); err != nil {
		log.Printf("unable to load env file %s: %v", envPath, err)
	}

	cfg := Config{
		Oauth2ClientID: os.Getenv("OAUTH2_CLIENT_ID"),
		Oauth2Secret:   os.Getenv("OAUTH2_SECRET"),
		ExternalURL:    os.Getenv("EXTERNAL_URL"),
		CssColumns:     os.Getenv("GBM_CSS_COLUMNS") != "",
		Namespace:      os.Getenv("GBM_NAMESPACE"),
		Provider:       os.Getenv("GBM_PROVIDER"),
	}

	configPath := DefaultConfigPath()

	var cfgFlag stringFlag
	var idFlag stringFlag
	var secretFlag stringFlag
	var urlFlag stringFlag
	var nsFlag stringFlag
	var providerFlag stringFlag
	var columnFlag boolFlag
	var versionFlag bool
	var dumpConfig bool
	flag.Var(&cfgFlag, "config", "path to config file")
	flag.Var(&idFlag, "client-id", "OAuth2 client ID")
	flag.Var(&secretFlag, "client-secret", "OAuth2 client secret")
	flag.Var(&urlFlag, "external-url", "external URL")
	flag.Var(&nsFlag, "namespace", "repository namespace")
	flag.Var(&providerFlag, "provider", fmt.Sprintf("git provider (%s)", strings.Join(ProviderNames(), ", ")))
	flag.Var(&columnFlag, "css-columns", "use CSS columns")
	flag.BoolVar(&versionFlag, "version", false, "show version")
	flag.BoolVar(&dumpConfig, "dump-config", false, "print merged config and exit")
	flag.Parse()

	if versionFlag {
		fmt.Printf("gobookmarks %s commit %s built %s\n", version, commit, date)
		fmt.Printf("providers: %s\n", strings.Join(ProviderNames(), ", "))
		return
	}

	if cfgFlag.set {
		configPath = cfgFlag.value
	}
	if fileCfg, err := LoadConfigFile(configPath); err == nil {
		MergeConfig(&cfg, fileCfg)
	} else {
		log.Printf("unable to load config file %s: %v", configPath, err)
	}

	if idFlag.set {
		cfg.Oauth2ClientID = idFlag.value
	}
	if secretFlag.set {
		cfg.Oauth2Secret = secretFlag.value
	}
	if urlFlag.set {
		cfg.ExternalURL = urlFlag.value
	}
	if nsFlag.set {
		cfg.Namespace = nsFlag.value
	}
	if columnFlag.set {
		cfg.CssColumns = columnFlag.value
	}
	if providerFlag.set {
		cfg.Provider = providerFlag.value
	}

	if dumpConfig {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		return
	}

	UseCssColumns = cfg.CssColumns
	Namespace = cfg.Namespace
	clientID = cfg.Oauth2ClientID
	clientSecret = cfg.Oauth2Secret
	externalUrl = cfg.ExternalURL

	if cfg.Provider != "" {
		if !SetProviderByName(cfg.Provider) {
			log.Fatalf("invalid provider %q. valid options: %s", cfg.Provider, strings.Join(ProviderNames(), ", "))
		}
	}

	SessionName = "gobookmarks"
	SessionStore = sessions.NewCookieStore([]byte("random-key")) // TODO random key
	if ActiveProvider == nil {
		fmt.Printf("no active provider please set one options: %v\n", ProviderNames())
		os.Exit(-1)
	}
	Oauth2Config = ActiveProvider.OAuth2Config(clientID, clientSecret, redirectUrl)

	r := mux.NewRouter()

	r.Use(UserAdderMiddleware)
	r.Use(CoreAdderMiddleware)

	r.HandleFunc("/main.css", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetMainCSSData())
	}).Methods("GET")
	r.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetFavicon())
	}).Methods("GET")

	// News
	r.Handle("/", http.HandlerFunc(runTemplate("indexPage.gohtml"))).Methods("GET")
	r.HandleFunc("/", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(HasError())
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditCreateAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Create"))
	r.HandleFunc("/edit", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/editCategory", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/editCategory", runHandlerChain(EditCategoryPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/editCategory", runHandlerChain(CategoryEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
	r.HandleFunc("/editCategory", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/history", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/history", runTemplate("history.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/history/commits", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/history/commits", runTemplate("historyCommits.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/logout", runHandlerChain(UserLogoutAction, runTemplate("logoutPage.gohtml"))).Methods("GET")

	r.HandleFunc("/proxy/favicon", FaviconProxyHandler).Methods("GET")

	http.Handle("/", r)

	if !fileExists("cert.pem") || !fileExists("key.pem") {
		CreatePEMFiles()
	}

	log.Printf("gobookmarks: %s, commit %s, built at %s", version, commit, date)
	SetVersion(version, commit, date)
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
					type ErrorData struct {
						*CoreData
						Error string
					}
					if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
						CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
						Error:    err.Error(),
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

func runTemplate(template string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type Data struct {
			*CoreData
			Error string
		}

		data := Data{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    r.URL.Query().Get("error"),
		}

		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, template, data); err != nil {
			log.Printf("Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
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
