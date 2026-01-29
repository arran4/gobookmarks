package app

import (
	"database/sql"

	"github.com/arran4/gobookmarks/core"
	"github.com/gorilla/sessions"
)

// App holds the application's long-lived dependencies.
type App struct {
	DB           *sql.DB
	SessionStore sessions.Store
	Repo         core.Repo
	UserProvider core.UserProvider
	Config       *Config
}

// NewApp creates a new App instance.
func NewApp(db *sql.DB, store sessions.Store, repo core.Repo, userProvider core.UserProvider, cfg *Config) *App {
	return &App{
		DB:           db,
		SessionStore: store,
		Repo:         repo,
		UserProvider: userProvider,
		Config:       cfg,
	}
}
