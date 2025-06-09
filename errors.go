package gobookmarks

import "errors"

// ErrRepoNotFound indicates that the bookmarks repository does not exist.
var ErrRepoNotFound = errors.New("repository not found")

// ErrHandled is returned by handlers when they have already written
// a response and no further handlers should run.
var ErrHandled = errors.New("handled")

// ErrSignedOut indicates that the OAuth token is no longer valid and
// the user must authenticate again.
var ErrSignedOut = errors.New("signed out")

// ErrNoProvider indicates that no provider was selected for the request.
var ErrNoProvider = errors.New("no provider selected")
