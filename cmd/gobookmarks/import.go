package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

type ImportCommand struct {
	*RootCommand
	fs   *flag.FlagSet
	Path string
	User string
}

func NewImportCommand(root *RootCommand) *ImportCommand {
	c := &ImportCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("import", flag.ExitOnError),
	}
	c.fs.StringVar(&c.Path, "path", "", "path to the bookmarks file")
	c.fs.StringVar(&c.User, "user", "", "user to import for (sql provider only)")
	return c
}

func (c *ImportCommand) Name() string {
	return c.fs.Name()
}

func (c *ImportCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *ImportCommand) Execute(args []string) error {
	c.fs.Parse(args)
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}

	provider, err := getConfiguredProvider(&c.RootCommand.cfg)
	if err != nil {
		return err
	}

	if c.User == "" && provider.Name() == "sql" {
		return fmt.Errorf("user is required for sql provider")
	}

	b, err := os.ReadFile(c.Path)
	if err != nil {
		return err
	}

	if err := provider.CreateBookmarks(context.Background(), c.User, nil, "main", string(b)); err != nil {
		return err
	}

	fmt.Printf("bookmarks imported from %s\n", c.Path)
	return nil
}
