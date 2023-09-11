package a4webbm

import (
	"bufio"
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
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

var (
	DbConnectionString   = os.Getenv("DB_CONNECTION_STRING")
	DbConnectionProvider = os.Getenv("DB_CONNECTION_PROVIDER")
)

func DBAdderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		db, err := sql.Open(DbConnectionProvider, DbConnectionString)
		if err != nil {
			log.Printf("error sql init: %s", err)
			http.Error(writer, "ERROR", 500)
			return
		}
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				log.Printf("Error closing db: %s", err)
			}
		}(db)
		ctx := request.Context()
		ctx = context.WithValue(ctx, ContextValues("sql.DB"), db)
		ctx = context.WithValue(ctx, ContextValues("queries"), New(db))
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
