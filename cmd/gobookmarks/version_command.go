package main

import (
	"flag"
	"fmt"
	"strings"

	. "github.com/arran4/gobookmarks"
)

type VersionCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Info   VersionInfo
}

func (rc *RootCommand) NewVersionCommand() (*VersionCommand, error) {
	c := &VersionCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("version", flag.ContinueOnError),
		Info:   rc.VersionInfo,
	}
	return c, nil
}

func (c *VersionCommand) Name() string {
	return c.Flags.Name()
}

func (c *VersionCommand) Parent() Command {
	return c.parent
}

func (c *VersionCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *VersionCommand) Subcommands() []Command {
	return nil
}

func (c *VersionCommand) Execute(args []string) error {
	if forwardHelpIfRequested(c, args) {
		return nil
	}
	if err := c.FlagSet().Parse(args); err != nil {
		c.FlagSet().Usage = func() {}
		printHelp(c, err)
		return err
	}
	fmt.Printf("gobookmarks %s commit %s built %s\n", c.Info.Version, c.Info.Commit, c.Info.Date)
	fmt.Printf("providers: %s\n", strings.Join(ProviderNames(), ", "))
	return nil
}
