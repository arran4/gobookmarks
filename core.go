package gobookmarks

import (
	"bufio"
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"github.com/arran4/gobookmarks/core"
)

func init() {
	gob.Register(&core.BasicUser{})
	gob.Register(&oauth2.Token{})
}

func CoreAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		session := request.Context().Value(core.ContextValues("session")).(*sessions.Session)
		githubUser, _ := session.Values["GithubUser"].(*core.BasicUser)
		providerName, _ := session.Values["Provider"].(string)

		login := ""
		if githubUser != nil {
			login = githubUser.Login
		}

		title := SiteTitle
		if title == "" {
			title = "gobookmarks"
		}
		if version == "dev" && !strings.HasPrefix(title, "dev: ") {
			title = "dev: " + title
		}

		ctx := context.WithValue(request.Context(), core.ContextValues("provider"), providerName)
		editMode := request.URL.Query().Get("edit") == "1"
		tab := TabFromRequest(request)
		ctx = context.WithValue(ctx, core.ContextValues("coreData"), &core.CoreData{
			UserRef:      login,
			Title:        title,
			EditMode:     editMode,
			Tab:          tab,
			RequestCache: &core.RequestCache{Data: make(map[string]*core.BookmarkCacheEntry)},
		})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

type Configuration struct {
	data map[string]string
}

// TODO use for settings
func NewConfiguration() *Configuration {
	return &Configuration{
		data: make(map[string]string),
	}
}

func (c *Configuration) set(key, value string) {
	c.data[key] = value
}

func (c *Configuration) get(key string) string {
	return c.data[key]
}

func (c *Configuration) readConfiguration(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("File close error: %s", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		sep := strings.Index(line, "=")
		if sep >= 0 {
			key := line[:sep]
			value := line[sep+1:]
			c.set(key, value)
		}
	}
}
