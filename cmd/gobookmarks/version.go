package main

import (
	"fmt"
	"strings"
	"flag"

	"github.com/arran4/gobookmarks"
)

func NewVersionCmd(version, commit, date string) *VersionCommand {
	return &VersionCommand{
		FlagSet: flag.NewFlagSet("version", flag.ExitOnError),
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

func (c *VersionCommand) Run(args []string) error {
	fmt.Printf("gobookmarks %s commit %s built %s\n", c.Version, c.Commit, c.Date)
	fmt.Printf("providers: %s\n", strings.Join(gobookmarks.ProviderNames(), ", "))
	return nil
}

func (c *VersionCommand) Usage() {
	printUsage(c.FlagSet, "version", "Print the version number of gobookmarks", `All software has versions. This is gobookmarks'`)
}
