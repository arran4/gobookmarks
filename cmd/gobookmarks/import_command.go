package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

type ImportCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Path   string
	User   string
}

func (rc *RootCommand) NewImportCommand() (*ImportCommand, error) {
	c := &ImportCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("import", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Path, "path", "", "path to the bookmarks file")
	c.Flags.StringVar(&c.User, "user", "", "user to import for (sql provider only)")
	return c, nil
}

func (c *ImportCommand) Name() string {
	return c.Flags.Name()
}

func (c *ImportCommand) Parent() Command {
	return c.parent
}

func (c *ImportCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ImportCommand) Subcommands() []Command {
	return nil
}

func (c *ImportCommand) Description() string {
	return "Import bookmarks into the configured provider"
}

func (c *ImportCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}
	if c.Path == "" {
		err := fmt.Errorf("path is required")
		printHelp(c, err)
		return err
	}

	provider, err := getConfiguredProvider(&c.parent.(*RootCommand).cfg)
	if err != nil {
		printHelp(c, err)
		return err
	}

	if c.User == "" && provider.Name() == "sql" {
		err := fmt.Errorf("user is required for sql provider")
		printHelp(c, err)
		return err
	}

	b, err := os.ReadFile(c.Path)
	if err != nil {
		printHelp(c, err)
		return err
	}

	if err := provider.CreateBookmarks(context.Background(), c.User, nil, "main", string(b)); err != nil {
		printHelp(c, err)
		return err
	}

	fmt.Printf("bookmarks imported from %s\n", c.Path)
	return nil
}
