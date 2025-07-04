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

// ErrUserExists indicates that a user already exists when attempting signup.
var ErrUserExists = errors.New("user already exists")

// ErrUserNotFound indicates that a user does not exist when attempting to set a password.
var ErrUserNotFound = errors.New("user not found")

// UserError wraps an error message intended for display to the user.
// It satisfies the error interface so it can be returned like a normal error.
// UserError describes an error that has a user facing message.
// The wrapped error can be inspected using errors.As/Is.
type UserError struct {
	Msg string // message shown to the user
	Err error  // underlying error
}

// Error implements the error interface by returning the underlying error
// message so logs and comparisons use the wrapped error.
func (e UserError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e UserError) Unwrap() error { return e.Err }

// NewUserError creates a UserError with the provided display message and
// underlying cause.
func NewUserError(msg string, err error) error {
	return UserError{Msg: msg, Err: err}
}

// SystemError represents a server-side failure that prevents the request
// from completing. The message is shown on a dedicated error page while the
// underlying error is logged for debugging.
type SystemError struct {
	Msg string // message shown to the user
	Err error  // underlying error
}

func (e SystemError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Err.Error()
}

func (e SystemError) Unwrap() error { return e.Err }

// NewSystemError creates a SystemError with the provided display message and
// underlying cause.
func NewSystemError(msg string, err error) error {
	return SystemError{Msg: msg, Err: err}
}
