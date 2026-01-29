package core

import "github.com/gorilla/sessions"

// Core defines the interface for the core data accessible to handlers.
// This allows gobookmarks to be embedded in other applications (like goa4web)
// that validly implement this interface.
type Core interface {
	GetSession() *sessions.Session
	GetUser() User
}

type User interface {
	GetLogin() string
}
