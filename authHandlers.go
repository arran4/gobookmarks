package a4webbm

import (
	"context"
	"github.com/google/go-github/v55/github"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

func UserLogoutAction(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		*CoreData
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
	}

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	delete(session.Values, "GithubUser")
	delete(session.Values, "Token")

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

	client := github.NewClient(oauth2.NewClient(r.Context(), oauth2.StaticTokenSource(token)))
	user, _, err := client.Users.Get(r.Context(), "")
	if err != nil {
		log.Printf("client.Users.Get error: %s", err)
		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "error.gohtml", ErrorData{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
			Error:    err.Error(),
		}); err != nil {
			log.Printf("Error client.Users.Get: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	session.Values["GithubUser"] = user
	session.Values["Token"] = token

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
