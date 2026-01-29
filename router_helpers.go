package gobookmarks

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

func runHandlerChain(chain ...any) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, each := range chain {
			switch each := each.(type) {
			case http.HandlerFunc:
				each(w, r)
			case http.Handler:
				each.ServeHTTP(w, r)
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
		tab := TabFromRequest(r)
		if v, ok := r.Context().Value(ContextValues("redirectTab")).(string); ok {
			if parsed, err := strconv.Atoi(v); err == nil {
				tab = parsed
			}
		}
		u.Path = TabPath(tab)
		page := r.PostFormValue("page")
		if v, ok := r.Context().Value(ContextValues("redirectPage")).(string); ok {
			page = v
		}
		if fragment := PageFragmentFromIndex(page); fragment != "" {
			u.Fragment = fragment
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
		u.Path = TabPath(TabFromRequest(r))
		if fragment := PageFragmentFromIndex(r.URL.Query().Get("page")); fragment != "" {
			u.Fragment = fragment
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
