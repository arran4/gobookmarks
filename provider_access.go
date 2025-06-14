package gobookmarks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type bookmarkCacheEntry struct {
	bookmarks string
	sha       string
	expiry    time.Time
}

var bookmarksCache = struct {
	sync.RWMutex
	data map[string]*bookmarkCacheEntry
}{data: make(map[string]*bookmarkCacheEntry)}

func cacheKey(user, ref string) string { return user + "|" + ref }

func getCachedBookmarks(user, ref string) (string, string, bool) {
	key := cacheKey(user, ref)
	bookmarksCache.RLock()
	entry, ok := bookmarksCache.data[key]
	bookmarksCache.RUnlock()
	if !ok || time.Now().After(entry.expiry) {
		return "", "", false
	}
	return entry.bookmarks, entry.sha, true
}

func setCachedBookmarks(user, ref, bookmarks, sha string) {
	key := cacheKey(user, ref)
	bookmarksCache.Lock()
	bookmarksCache.data[key] = &bookmarkCacheEntry{bookmarks: bookmarks, sha: sha, expiry: time.Now().Add(time.Minute)}
	bookmarksCache.Unlock()
}

func invalidateBookmarkCache(user string) {
	bookmarksCache.Lock()
	for k := range bookmarksCache.data {
		if strings.HasPrefix(k, user+"|") {
			delete(bookmarksCache.data, k)
		}
	}
	bookmarksCache.Unlock()
}

func providerFromContext(ctx context.Context) Provider {
	if name, ok := ctx.Value(ContextValues("provider")).(string); ok {
		if p := GetProvider(name); p != nil {
			return p
		}
	}
	return nil
}

type ProviderCreds struct {
	ID     string
	Secret string
}

func providerCreds(name string) *ProviderCreds {
	switch name {
	case "github":
		if GithubClientID == "" || GithubClientSecret == "" {
			return nil
		}
		return &ProviderCreds{ID: GithubClientID, Secret: GithubClientSecret}
	case "gitlab":
		if GitlabClientID == "" || GitlabClientSecret == "" {
			return nil
		}
		return &ProviderCreds{ID: GitlabClientID, Secret: GitlabClientSecret}
	case "git":
		if LocalGitPath == "" {
			return nil
		}
		return &ProviderCreds{}
	default:
		return nil
	}
}

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	tags, err := p.GetTags(ctx, user, token)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return nil, ErrSignedOut
	}
	return tags, err
}

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	bs, err := p.GetBranches(ctx, user, token)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return nil, ErrSignedOut
	}
	return bs, err
}

func GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	cs, err := p.GetCommits(ctx, user, token)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return nil, ErrSignedOut
	}
	return cs, err
}

func GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	if b, sha, ok := getCachedBookmarks(user, ref); ok {
		return b, sha, nil
	}
	p := providerFromContext(ctx)
	if p == nil {
		return "", "", ErrNoProvider
	}
	b, sha, err := p.GetBookmarks(ctx, user, ref, token)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return "", "", ErrSignedOut
	}
	return b, sha, err
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	err := p.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
	if err == nil {
		invalidateBookmarkCache(user)
	} else if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return ErrSignedOut
	}
	return err
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	err := p.CreateBookmarks(ctx, user, token, branch, text)
	if err == nil {
		invalidateBookmarkCache(user)
	} else if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return ErrSignedOut
	}
	return err
}
