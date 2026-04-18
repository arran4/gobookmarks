package main

import (
	"errors"

	gobookmarks "github.com/arran4/gobookmarks"
)

func getConfiguredProvider(cfg *gobookmarks.Configuration) (gobookmarks.Provider, error) {
	if cfg.DBConnectionProvider != "" && cfg.DBConnectionString != "" {
		return &gobookmarks.SQLProvider{}, nil
	}
	if cfg.LocalGitPath != "" {
		return gobookmarks.GitProvider{}, nil
	}
	// This is a simplification. A real implementation would need to handle
	// OAuth2 providers, but that's outside the scope of this task.
	return nil, errors.New("no provider configured")
}
