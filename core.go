package gobookmarks

import (
	"bufio"
	"context"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
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

		title := SiteTitle
		if title == "" {
			title = "Arran4's Bookmarks Website"
		}
		if version == "dev" && !strings.HasPrefix(title, "dev: ") {
			title = "dev: " + title
		}

		ctx := context.WithValue(request.Context(), ContextValues("provider"), providerName)
		ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{
			UserRef: login,
			Title:   title,
		})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

type CoreData struct {
	Title       string
	AutoRefresh bool
	UserRef     string
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

type ContextValues string
