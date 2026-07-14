package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// TargetScope determines whether the skill is installed globally for the user or locally for the project.
type TargetScope string

const (
	ScopeUser    TargetScope = "user"
	ScopeProject TargetScope = "project"
)

// AgentTarget represents an AI agent that can have skills installed.
type AgentTarget interface {
	Name() string
	InstallDir(scope TargetScope) (string, error)
}

type commonAgent struct{}

func (c *commonAgent) Name() string { return "common" }

func (c *commonAgent) InstallDir(scope TargetScope) (string, error) {
	if scope == ScopeProject {
		return filepath.Abs(".agents/skills")
	}
	// For user scope, try XDG Config Home, then fallback to ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "agents", "skills"), nil
}

type cursorAgent struct{}

func (c *cursorAgent) Name() string { return "cursor" }

func (c *cursorAgent) InstallDir(scope TargetScope) (string, error) {
	if scope == ScopeProject {
		return filepath.Abs(".cursor/rules")
	}
	// Note: cursor's actual global rule location varies, fallback to common pattern for user
	return (&commonAgent{}).InstallDir(scope)
}

type copilotAgent struct{}

func (c *copilotAgent) Name() string { return "copilot" }

func (c *copilotAgent) InstallDir(scope TargetScope) (string, error) {
	if scope == ScopeProject {
		return filepath.Abs(".github/copilot-instructions")
	}
	return (&commonAgent{}).InstallDir(scope)
}


// GetAgentTarget returns an AgentTarget implementation by name.
func GetAgentTarget(name string) (AgentTarget, error) {
	switch name {
	case "common", "":
		return &commonAgent{}, nil
	case "cursor":
		return &cursorAgent{}, nil
	case "copilot":
		return &copilotAgent{}, nil
	default:
		return nil, fmt.Errorf("unsupported agent target: %s", name)
	}
}
