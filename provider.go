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
	OAuth2Config(clientID, clientSecret, redirectURL string) *oauth2.Config
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

var ActiveProvider Provider

// SetProviderByName activates the named provider.
// It returns true if the provider exists.
func SetProviderByName(name string) bool {
	if p, ok := providers[name]; ok {
		prevDefault := ""
		if ActiveProvider != nil {
			prevDefault = ActiveProvider.DefaultServer()
		}
		ActiveProvider = p
		if GitServer == "" || GitServer == prevDefault {
			GitServer = p.DefaultServer()
		}
		return true
	}
	return false
}

func init() {
	if p, ok := providers["github"]; ok {
		ActiveProvider = p
		if GitServer == "" {
			GitServer = p.DefaultServer()
		}
	}
}
