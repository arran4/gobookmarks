package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arran4/gobookmarks/skill"
)

type SkillInstallCommand struct {
	parent Command
	Flags  *flag.FlagSet

	Scope string
	Agent string
}

func (sc *SkillCommand) NewSkillInstallCommand() *SkillInstallCommand {
	c := &SkillInstallCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("install", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	return c
}

func (c *SkillInstallCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillInstallCommand) Parent() Command {
	return c.parent
}

func (c *SkillInstallCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillInstallCommand) Subcommands() []Command {
	return nil
}

func (c *SkillInstallCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	remaining := c.FlagSet().Args()
	if len(remaining) < 1 {
		err := fmt.Errorf("missing source argument\nUsage: gobookmarks skill install <source> [skill-name]")
		printHelp(c, err)
		return err
	}

	sourceStr := remaining[0]

	// Determine skill name
	name := ""
	if len(remaining) >= 2 {
		name = remaining[1]
	} else {
		// Infer from source
		parts := strings.Split(sourceStr, "/")
		if len(parts) > 0 {
			name = parts[len(parts)-1]
			name = strings.TrimSuffix(name, ".git")
		}
	}

	if name == "" {
		return fmt.Errorf("could not infer skill name from source, please provide it explicitly")
	}

	source, err := skill.ParseSource(sourceStr, name)
	if err != nil {
		return err
	}

	scope := skill.TargetScope(c.Scope)
	if scope != skill.ScopeUser && scope != skill.ScopeProject {
		return fmt.Errorf("invalid scope %q (must be 'user' or 'project')", c.Scope)
	}

	manager := skill.NewSkillManager()
	fmt.Printf("Installing skill '%s' for agent '%s' (%s scope)...\n", name, c.Agent, c.Scope)

	err = manager.Install(context.Background(), source, name, c.Agent, scope)
	if err != nil {
		return err
	}

	target, _ := skill.GetAgentTarget(c.Agent)
	dir, _ := target.InstallDir(scope)

	fmt.Printf("Successfully installed skill '%s' to %s\n", name, filepath.Join(dir, name))
	return nil
}
