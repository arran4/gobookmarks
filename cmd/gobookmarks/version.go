package main

import (
	"flag"
	"fmt"
	"strings"

	. "github.com/arran4/gobookmarks"
)

type VersionCommand struct {
	*RootCommand
	fs *flag.FlagSet
}

func NewVersionCommand(root *RootCommand) *VersionCommand {
	c := &VersionCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("version", flag.ExitOnError),
	}
	return c
}

func (c *VersionCommand) Name() string {
	return c.fs.Name()
}

func (c *VersionCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *VersionCommand) Execute(args []string) error {
	fmt.Printf("gobookmarks %s commit %s built %s\n", version, commit, date)
	fmt.Printf("providers: %s\n", strings.Join(ProviderNames(), ", "))
	return nil
}
