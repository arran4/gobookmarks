package main

import (
	"fmt"
	"strings"

	"github.com/arran4/gobookmarks"
)

func (c *VersionCommand) Name() string  { return "version" }
func (c *VersionCommand) Short() string { return "Print the version number of gobookmarks" }
func (c *VersionCommand) Long() string {
	return `All software has versions. This is gobookmarks'`
}

func (c *VersionCommand) Run(args []string) error {
	fmt.Printf("gobookmarks %s commit %s built %s\n", c.Version, c.Commit, c.Date)
	fmt.Printf("providers: %s\n", strings.Join(gobookmarks.ProviderNames(), ", "))
	return nil
}
