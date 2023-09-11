package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"time"
)

var (
	clientID                    = os.Getenv("OAUTH2_CLIENT_ID")
	clientSecret                = os.Getenv("OAUTH2_SECRET")
	externalUrl                 = os.Getenv("EXTERNAL_URL")
	redirectUrl                 = fmt.Sprintf("%s/oauth2Callback", externalUrl)
	sessionName                 = "a4webbookmarks"
	store                       = sessions.NewCookieStore([]byte("random-key"))
	oauth2Config *oauth2.Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     endpoints.Google,
	}
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func main() {
	r := mux.NewRouter()

	r.Use(DBAdderMiddleware)
	r.Use(UserAdderMiddleware)
	r.Use(CoreAdderMiddleware)

	r.HandleFunc("/main.css", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(getMainCSSData())
	}).Methods("GET")

	// News
	r.Handle("/", http.HandlerFunc(runTemplate("indexPage.gohtml"))).Methods("GET")
	r.Handle("/", http.HandlerFunc(taskDoneAutoRefreshPage)).Methods("POST")

	bmr := r.PathPrefix("/bookmarks").Subrouter()
	bmr.HandleFunc("", runTemplate("bookmarksPage.gohtml")).Methods("GET")
	bmr.HandleFunc("/mine", runTemplate("bookmarksMinePage.gohtml")).Methods("GET", "POST")
	bmr.HandleFunc("/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	bmr.HandleFunc("/edit", runTemplate("bookmarksEditPage.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())
	bmr.HandleFunc("/edit", runHandlerChain(bookmarksEditSaveAction, redirectToHandler("/bookmarks/mine"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
	bmr.HandleFunc("/edit", runHandlerChain(bookmarksEditCreateAction, redirectToHandler("/bookmarks/mine"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Create"))
	bmr.HandleFunc("/edit", taskDoneAutoRefreshPage).Methods("POST")

	r.HandleFunc("/logout", runHandlerChain(userLogoutAction, runTemplate("userLogoutPage.gohtml"))).Methods("GET")
	r.HandleFunc("/oauth2Callback", runHandlerChain(oauth2CallbackPage, redirectToHandler("/bookmarks/mine"))).Methods("GET")

	http.Handle("/", r)

	if !fileExists("cert.pem") || !fileExists("key.pem") {
		CreatePEMFiles()
	}

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
		}

		data := Data{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		}

		if err := getCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, template, data); err != nil {
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

func RequiresAnAccount() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		var session *sessions.Session
		sessioni := request.Context().Value(ContextValues("session"))
		if sessioni == nil {
			var err error
			session, err = store.Get(request, sessionName)
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
		userRef, _ := session.Values["UserRef"].(string)
		return userRef != ""
	}
}

func TaskMatcher(taskName string) mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.PostFormValue("task") == taskName
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
