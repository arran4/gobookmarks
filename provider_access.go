package gobookmarks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/arran4/gobookmarks/core"
)

var bookmarksCache = struct {
	sync.RWMutex
	Data map[string]*core.BookmarkCacheEntry
}{Data: make(map[string]*core.BookmarkCacheEntry)}

func cacheKey(user, ref string) string { return user + "|" + ref }

func invalidateRequestCache(ctx context.Context, user string) {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.RequestCache != nil {
		cd.RequestCache.Lock()
		defer cd.RequestCache.Unlock()
		prefix := user + "|"
		for k := range cd.RequestCache.Data {
			if strings.HasPrefix(k, prefix) {
				delete(cd.RequestCache.Data, k)
			}
		}
	}
}

func getCachedBookmarks(user, ref string) (string, string, bool) {
	key := cacheKey(user, ref)
	bookmarksCache.RLock()
	entry, ok := bookmarksCache.Data[key]
	bookmarksCache.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return "", "", false
	}
	return entry.Bookmarks, entry.SHA, true
}

func setCachedBookmarks(user, ref, bookmarks, sha string) {
	key := cacheKey(user, ref)
	bookmarksCache.Lock()
	bookmarksCache.Data[key] = &core.BookmarkCacheEntry{Bookmarks: bookmarks, SHA: sha, Expiry: time.Now().Add(time.Minute)}
	bookmarksCache.Unlock()
}

func invalidateBookmarkCache(user string) {
	bookmarksCache.Lock()
	for k := range bookmarksCache.Data {
		if strings.HasPrefix(k, user+"|") {
			delete(bookmarksCache.Data, k)
		}
	}
	bookmarksCache.Unlock()
}

func providerFromContext(ctx context.Context) Provider {
	if name, ok := ctx.Value(core.ContextValues("provider")).(string); ok {
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
	case "sql":
		if DBConnectionProvider == "" {
			return nil
		}
		return &ProviderCreds{}
	default:
		return nil
	}
}

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*core.Tag, error) {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		return cd.Repo.GetTags(ctx, user, token)
	}
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

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*core.Branch, error) {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		return cd.Repo.GetBranches(ctx, user, token)
	}
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

func GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*core.Commit, error) {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		return cd.Repo.GetCommits(ctx, user, token, ref, page, perPage)
	}
	p := providerFromContext(ctx)
	if p == nil {
		return nil, ErrNoProvider
	}
	cs, err := p.GetCommits(ctx, user, token, ref, page, perPage)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return nil, ErrSignedOut
	}
	return cs, err
}

func GetAdjacentCommits(ctx context.Context, user string, token *oauth2.Token, ref, sha string) (string, string, error) {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		return cd.Repo.AdjacentCommits(ctx, user, token, ref, sha)
	}
	p := providerFromContext(ctx)
	if p == nil {
		return "", "", ErrNoProvider
	}
	if ap, ok := p.(AdjacentCommitProvider); ok {
		prev, next, err := ap.AdjacentCommits(ctx, user, token, ref, sha)
		if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
			return "", "", ErrSignedOut
		}
		return prev, next, err
	}
	return "", "", nil
}

func GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	key := cacheKey(user, ref)
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.RequestCache != nil {
		cd.RequestCache.RLock()
		if entry, ok := cd.RequestCache.Data[key]; ok {
			cd.RequestCache.RUnlock()
			return entry.Bookmarks, entry.SHA, nil
		}
		cd.RequestCache.RUnlock()
	}

	if b, sha, ok := getCachedBookmarks(user, ref); ok {
		return b, sha, nil
	}
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		b, sha, err := cd.Repo.GetBookmarks(ctx, user, ref, token)
		if err == nil && cd.RequestCache != nil {
			cd.RequestCache.Lock()
			cd.RequestCache.Data[key] = &core.BookmarkCacheEntry{Bookmarks: b, SHA: sha}
			cd.RequestCache.Unlock()
		}
		return b, sha, err
	}
	p := providerFromContext(ctx)
	if p == nil {
		return "", "", ErrNoProvider
	}
	b, sha, err := p.GetBookmarks(ctx, user, ref, token)
	if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return "", "", ErrSignedOut
	}
	if err == nil {
		if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.RequestCache != nil {
			cd.RequestCache.Lock()
			cd.RequestCache.Data[key] = &core.BookmarkCacheEntry{Bookmarks: b, SHA: sha}
			cd.RequestCache.Unlock()
		}
	}
	return b, sha, err
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		err := cd.Repo.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
		if err == nil {
			invalidateBookmarkCache(user)
			invalidateRequestCache(ctx, user)
		}
		return err
	}
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	err := p.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
	if err == nil {
		invalidateBookmarkCache(user)
		invalidateRequestCache(ctx, user)
	} else if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return ErrSignedOut
	}
	return err
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if cd, ok := ctx.Value(core.ContextValues("coreData")).(*core.CoreData); ok && cd.Repo != nil {
		err := cd.Repo.CreateBookmarks(ctx, user, token, branch, text)
		if err == nil {
			invalidateBookmarkCache(user)
			invalidateRequestCache(ctx, user)
		}
		return err
	}
	p := providerFromContext(ctx)
	if p == nil {
		return ErrNoProvider
	}
	err := p.CreateBookmarks(ctx, user, token, branch, text)
	if err == nil {
		invalidateBookmarkCache(user)
		invalidateRequestCache(ctx, user)
	} else if errors.Is(err, ErrRepoNotFound) && p.Name() == "git" {
		return ErrSignedOut
	}
	return err
}
