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

func SetProviderByName(name string) {
	if p, ok := providers[name]; ok {
		ActiveProvider = p
	}
}

func init() {
	if p, ok := providers["github"]; ok {
		ActiveProvider = p
	}
}
