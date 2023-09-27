package a4webbm

import (
	"bufio"
	"context"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"os"
	"strings"
)

func CoreAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		session := request.Context().Value(ContextValues("session")).(*sessions.Session)
		userRef, _ := session.Values["UserRef"].(string)

		ctx := context.WithValue(request.Context(), ContextValues("coreData"), &CoreData{
			UserRef: userRef,
			Title:   "Arran4's Bookmarks Website",
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
