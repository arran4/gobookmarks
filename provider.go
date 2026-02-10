package gobookmarks

import (
	"context"
	"time"

	"golang.org/x/oauth2"
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
	Config(ctx context.Context, clientID, clientSecret, redirectURL string) *oauth2.Config
	CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error)
	GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error)
	GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error)
	GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error)
	GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error)
	UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error
	CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error
	CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error
	RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error)
	DefaultServer() string
}

// AdjacentCommitProvider optionally provides methods to navigate commit history.
type AdjacentCommitProvider interface {
	AdjacentCommits(ctx context.Context, user string, token *oauth2.Token, ref, sha string) (string, string, error)
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

var (
	providers     = map[string]Provider{}
	providerOrder []string
)

// RegisterProvider registers a Provider by name. The order providers are
// registered is tracked but the final ordering returned by ProviderNames may
// be modified via SetProviderOrder.
func RegisterProvider(p Provider) {
	name := p.Name()
	providers[name] = p
	for _, n := range providerOrder {
		if n == name {
			return
		}
	}
	providerOrder = append(providerOrder, name)
}

func GetDefaultProviderOrder() []string {
	names := make([]string, len(providerOrder))
	copy(names, providerOrder)
	return names
}

func GetProvider(name string) Provider { return providers[name] }
