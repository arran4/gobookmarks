package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/arran4/gobookmarks/skill"
)

type SkillListCommand struct {
	parent Command
	Flags  *flag.FlagSet

	Scope string
	Agent string
}

func (sc *SkillCommand) NewSkillListCommand() *SkillListCommand {
	c := &SkillListCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("list", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	return c
}

func (c *SkillListCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillListCommand) Parent() Command {
	return c.parent
}

func (c *SkillListCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillListCommand) Subcommands() []Command {
	return nil
}

func (c *SkillListCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	manager := skill.NewSkillManager()
	scope := skill.TargetScope(c.Scope)

	skills, err := manager.List(c.Agent, scope)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Printf("No skills installed for agent '%s' (%s scope).\n", c.Agent, c.Scope)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tREVISION")
	for _, s := range skills {
		rev := s.Revision
		if len(rev) > 8 {
			rev = rev[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Source, rev)
	}
	w.Flush()

	return nil
}
