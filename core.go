package gobookmarks

import (
	"context"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

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
	Title       string
	AutoRefresh bool
	UserRef     string
	EditMode    bool
	Tab         int
	requestCache *requestCache
}

type ContextValues string
