package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
	"time"
)

type User struct {
	Login string
}

type Branch struct {
	Name string
}

type Tag struct {
	Name string
}

type Commit struct {
	SHA            string
	Message        string
	CommitterName  string
	CommitterEmail string
	CommitterDate  time.Time
}

type Provider interface {
	Name() string
	Config(clientID, clientSecret, redirectURL string) *oauth2.Config
	CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error)
	GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error)
	GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error)
	GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error)
	UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error
	DefaultServer() string
}

// PasswordHandler is implemented by providers that manage passwords.
// PasswordHandler manages user accounts for providers that do not rely on
// external authentication.
//
// CreateUser registers a new account and returns ErrUserExists if the user is
// already present. SetPassword updates the password for an existing user and
// returns ErrUserNotFound when the account does not exist.
type PasswordHandler interface {
	CreateUser(ctx context.Context, user, password string) error
	SetPassword(ctx context.Context, user, password string) error
	CheckPassword(ctx context.Context, user, password string) (bool, error)
}

var providers = map[string]Provider{}

func RegisterProvider(p Provider) { providers[p.Name()] = p }

func GetProvider(name string) Provider { return providers[name] }

func ProviderNames() []string {
	names := make([]string, 0, len(providers))
	for n := range providers {
		names = append(names, n)
	}
	return names
}
