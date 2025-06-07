package gobookmarks

import "errors"

// ErrRepoNotFound indicates that the bookmarks repository does not exist.
var ErrRepoNotFound = errors.New("repository not found")

// ErrHandled is returned by handlers when they have already written
// a response and no further handlers should run.
var ErrHandled = errors.New("handled")
