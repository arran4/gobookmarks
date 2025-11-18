package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

type ExportCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Path   string
	User   string
}

func (rc *RootCommand) NewExportCommand() (*ExportCommand, error) {
	c := &ExportCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("export", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Path, "path", "", "path to export the bookmarks to")
	c.Flags.StringVar(&c.User, "user", "", "user to export for (sql provider only)")
	return c, nil
}

func (c *ExportCommand) Name() string {
	return c.Flags.Name()
}

func (c *ExportCommand) Parent() Command {
	return c.parent
}

func (c *ExportCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ExportCommand) Subcommands() []Command {
	return nil
}

func (c *ExportCommand) Description() string {
	return "Export bookmarks from the configured provider"
}

func (c *ExportCommand) Execute(args []string) error {
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

	data, err := provider.GetBookmarks(context.Background(), c.User, nil, "main")
	if err != nil {
		printHelp(c, err)
		return err
	}

	if err := os.WriteFile(c.Path, []byte(data), 0644); err != nil {
		printHelp(c, err)
		return err
	}

	fmt.Printf("bookmarks exported to %s\n", c.Path)
	return nil
}
