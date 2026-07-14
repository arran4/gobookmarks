package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/arran4/gobookmarks/skill"
)

type SkillInspectCommand struct {
	parent Command
	Flags  *flag.FlagSet

	Scope string
	Agent string
}

func (sc *SkillCommand) NewSkillInspectCommand() *SkillInspectCommand {
	c := &SkillInspectCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("inspect", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	return c
}

func (c *SkillInspectCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillInspectCommand) Parent() Command {
	return c.parent
}

func (c *SkillInspectCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillInspectCommand) Subcommands() []Command {
	return nil
}

func (c *SkillInspectCommand) Execute(args []string) error {
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
		err := fmt.Errorf("missing skill name argument\nUsage: gobookmarks skill inspect <name>")
		printHelp(c, err)
		return err
	}

	name := remaining[0]
	scope := skill.TargetScope(c.Scope)

	target, err := skill.GetAgentTarget(c.Agent)
	if err != nil {
		return err
	}

	installDir, err := target.InstallDir(scope)
	if err != nil {
		return err
	}

	skillDir := filepath.Join(installDir, name)
	md, err := skill.ReadMetadata(skillDir)
	if err != nil {
		return fmt.Errorf("failed to read metadata for skill '%s': %w", name, err)
	}
	if md == nil {
		return fmt.Errorf("skill '%s' not found or missing metadata", name)
	}

	b, _ := json.MarshalIndent(md, "", "  ")
	fmt.Printf("Skill metadata for '%s':\n%s\n", name, string(b))
	fmt.Printf("\nInstallation path: %s\n", skillDir)
	return nil
}
