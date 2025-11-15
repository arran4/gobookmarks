package main

import (
	"flag"
)

// ServeCommand is the serve command.
type ServeCommand struct {
	FlagSet *flag.FlagSet
	Version string
	Commit  string
	Date    string
}

// VersionCommand is the version command.
type VersionCommand struct {
	FlagSet *flag.FlagSet
	Version string
	Commit  string
	Date    string
}

// VerifyFileCommand is the verify-file command.
type VerifyFileCommand struct {
	FlagSet *flag.FlagSet
}

// VerifyCredsCommand is the verify-creds command.
type VerifyCredsCommand struct {
	FlagSet *flag.FlagSet
}

// DbCommand is the db command.
type DbCommand struct {
	FlagSet *flag.FlagSet
}

// GitCommand is the git command.
type GitCommand struct {
	FlagSet *flag.FlagSet
}
