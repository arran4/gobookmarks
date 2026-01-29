package gobookmarks

import (
	"context"
	"sort"

	"github.com/arran4/gobookmarks/core"

	"golang.org/x/oauth2"
)

// Types moved to core package, used directly now.

type Provider interface {
	Name() string
	Config(clientID, clientSecret, redirectURL string) *oauth2.Config
	// CurrentUser returns the currently authenticated user
	CurrentUser(ctx context.Context, token *oauth2.Token) (core.User, error)
	GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*core.Tag, error)
	GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*core.Branch, error)
	GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*core.Commit, error)
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

// SetProviderOrder updates the order in which providers are returned by
// ProviderNames. Names not recognized are ignored. Any registered providers not
// mentioned remain at the end in alphabetical order.
func SetProviderOrder(order []string) {
	if len(order) == 0 {
		names := make([]string, 0, len(providers))
		for n := range providers {
			names = append(names, n)
		}
		sort.Strings(names)
		providerOrder = names
		return
	}

	seen := make(map[string]bool)
	var final []string
	for _, n := range order {
		if _, ok := providers[n]; ok && !seen[n] {
			final = append(final, n)
			seen[n] = true
		}
	}
	var rest []string
	for n := range providers {
		if !seen[n] {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	providerOrder = append(final, rest...)
}

func GetProvider(name string) Provider { return providers[name] }

// ProviderNames returns the list of registered provider names in the order set
// by SetProviderOrder. The returned slice should not be modified.
func ProviderNames() []string {
	names := make([]string, len(providerOrder))
	copy(names, providerOrder)
	return names
}

// ConfiguredProviderNames returns the list of providers that are both
// compiled in and configured for use. A provider is considered configured
// when providerCreds returns non-nil.
func ConfiguredProviderNames() []string {
	names := make([]string, 0, len(providers))
	for _, n := range ProviderNames() {
		if providerCreds(n) != nil {
			names = append(names, n)
		}
	}
	return names
}
