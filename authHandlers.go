package a4webbm

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"strings"
)

func UserLogoutAction(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		*CoreData
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
	}

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	delete(session.Values, "UserRef")
	delete(session.Values, "AccessToken")
	delete(session.Values, "RefreshToken")

	if err := session.Save(r, w); err != nil {
		log.Printf("session.Save Error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data.CoreData.UserRef = ""
}

var (
	Oauth2Config *oauth2.Config
	SessionStore sessions.Store
	SessionName  string
)

func Oauth2CallbackPage(w http.ResponseWriter, r *http.Request) {

	type ErrorData struct {
		*CoreData
		Error string
	}

	token, err := Oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("Exchange error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	session, err := SessionStore.Get(r, SessionName)
	if err != nil {
		log.Printf("Session error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	provider, err := oidc.NewProvider(r.Context(), "https://accounts.google.com")
	if err != nil {
		log.Printf("oidc new provider error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	var verifier = provider.Verifier(&oidc.Config{ClientID: Oauth2Config.ClientID})

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Printf("id_otken missing")
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    "id token missing",
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("Id token failed to verify error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("IdToken claims error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	session.Values["UserRef"] = strings.ToLower(claims.Email)
	session.Values["AccessToken"] = token.AccessToken
	session.Values["RefreshToken"] = token.RefreshToken

	if err := session.Save(r, w); err != nil {
		log.Printf("Exchange error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
}

func UserAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Get the session.
		session, err := SessionStore.Get(request, SessionName)
		if err != nil {
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(request.Context(), ContextValues("session"), session)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
