package main

import (
	"fmt"

	. "github.com/arran4/gobookmarks"
)

func getConfiguredProvider(cfg *Configuration) (Provider, error) {
	if cfg.DBConnectionProvider != "" && cfg.DBConnectionString != "" {
		return SQLProvider{}, nil
	}
	if cfg.LocalGitPath != "" {
		return GitProvider{}, nil
	}
	// This is a simplification. A real implementation would need to handle
	// OAuth2 providers, but that's outside the scope of this task.
	return nil, fmt.Errorf("no provider configured")
}
