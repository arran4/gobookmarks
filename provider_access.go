package gobookmarks

import (
	"context"
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

func GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	return ActiveProvider.GetTags(ctx, user, token)
}

func GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	return ActiveProvider.GetBranches(ctx, user, token)
}

func GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	return ActiveProvider.GetCommits(ctx, user, token)
}

func GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	if b, sha, ok := getCachedBookmarks(user, ref); ok {
		return b, sha, nil
	}
	b, sha, err := ActiveProvider.GetBookmarks(ctx, user, ref, token)
	if err == nil {
		setCachedBookmarks(user, ref, b, sha)
	}
	return b, sha, err
}

func UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	err := ActiveProvider.UpdateBookmarks(ctx, user, token, sourceRef, branch, text, expectSHA)
	if err == nil {
		invalidateBookmarkCache(user)
	}
	return err
}

func CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	err := ActiveProvider.CreateBookmarks(ctx, user, token, branch, text)
	if err == nil {
		invalidateBookmarkCache(user)
	}
	return err
}
