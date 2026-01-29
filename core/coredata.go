package core

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

// CoreData represents the request-scoped context data.
type CoreData struct {
	User        User
	Repo        Repo
	EditMode    bool
	Tab         int
	Title       string
	AutoRefresh bool
	UserRef     string // For backward compatibility if needed, or derived from User
	Session     *sessions.Session
	// Add other request-scoped fields as needed
	RequestCache *RequestCache
}

func (cd *CoreData) GetSession() *sessions.Session {
	return cd.Session
}

type RequestCache struct {
	sync.RWMutex
	Data map[string]*BookmarkCacheEntry
}

type BookmarkCacheEntry struct {
	Bookmarks string
	SHA       string
	Expiry    time.Time
}

// BasicUser represents the authenticated user.
type BasicUser struct {
	Login string
}

func (u *BasicUser) GetLogin() string {
	return u.Login
}

func (cd *CoreData) GetUser() User {
	// If User is nil, we return nil (or typed nil which is fine as interface value usually checks nil)
	// However, usually we return the interface.
	if cd.User == nil {
		return nil
	}
	return cd.User
}

// UserProvider defines the interface for external user management
type UserProvider interface {
	// CurrentUser returns the current user from the request context
	CurrentUser(r *http.Request) (User, error)
	IsLoggedIn(r *http.Request) bool
}

// Repo defines the interface for data access.
type Repo interface {
	GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error)
	UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error)
	CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error
	CreateUser(ctx context.Context, user, password string) error
	CheckPassword(ctx context.Context, user, password string) (bool, error)
	GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error)
	GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error)
	AdjacentCommits(ctx context.Context, user string, token *oauth2.Token, ref, sha string) (string, string, error)
}

// Type definitions for params (moved from top level or redefined)
type Branch struct {
	Name string
}

type Commit struct {
	SHA            string
	Message        string
	CommitterName  string
	CommitterEmail string
	CommitterDate  time.Time
}

type Tag struct {
	Name string
}

// SessionManager abstract session operations if needed
type SessionManager interface {
	Get(r any, name string) (*sessions.Session, error)
}

type ContextValues string
