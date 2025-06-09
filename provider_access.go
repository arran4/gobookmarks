package gobookmarks

import (
	"context"
	"golang.org/x/oauth2"
)

func providerFromContext(ctx context.Context) Provider {
	if name, ok := ctx.Value(ContextValues("provider")).(string); ok {
		if p := GetProvider(name); p != nil {
			return p
		}
	}
	return nil
}

func providerCreds(name string) (clientID, secret string) {
	switch name {
	case "github":
		return GithubClientID, GithubClientSecret
	case "gitlab":
		return GitlabClientID, GitlabClientSecret
	default:
		return "", ""
	}
}

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	return p.GetTags(ctx, user, token)
}

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	return p.GetBranches(ctx, user, token)
}

func GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	return p.GetCommits(ctx, user, token)
}

func GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return "", "", ErrNoProvider
	}
	return p.GetBookmarks(ctx, user, ref, token)
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	return p.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	return p.CreateBookmarks(ctx, user, token, branch, text)
}
