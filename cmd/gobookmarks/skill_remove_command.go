package main

import (
	"flag"
	"fmt"
	"github.com/arran4/gobookmarks/skill"
)

type SkillRemoveCommand struct {
	parent Command
	Flags  *flag.FlagSet

	Scope string
	Agent string
}

func (sc *SkillCommand) NewSkillRemoveCommand() *SkillRemoveCommand {
	c := &SkillRemoveCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("remove", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	return c
}

func (c *SkillRemoveCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillRemoveCommand) Parent() Command {
	return c.parent
}

func (c *SkillRemoveCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillRemoveCommand) Subcommands() []Command {
	return nil
}

func (c *SkillRemoveCommand) Execute(args []string) error {
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
		err := fmt.Errorf("missing skill name argument\nUsage: gobookmarks skill remove <name>")
		printHelp(c, err)
		return err
	}

	name := remaining[0]
	manager := skill.NewSkillManager()
	scope := skill.TargetScope(c.Scope)

	err := manager.Remove(name, c.Agent, scope)
	if err != nil {
		return err
	}

	fmt.Printf("Removed skill '%s' (agent: %s, scope: %s)\n", name, c.Agent, c.Scope)
	return nil
}
