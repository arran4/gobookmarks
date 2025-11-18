package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

type ExportCommand struct {
	*RootCommand
	fs   *flag.FlagSet
	Path string
	User string
}

func NewExportCommand(root *RootCommand) *ExportCommand {
	c := &ExportCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("export", flag.ExitOnError),
	}
	c.fs.StringVar(&c.Path, "path", "", "path to the bookmarks file")
	c.fs.StringVar(&c.User, "user", "", "user to export for (sql provider only)")
	return c
}

func (c *ExportCommand) Name() string {
	return c.fs.Name()
}

func (c *ExportCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *ExportCommand) Execute(args []string) error {
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

	bookmarks, _, err := provider.GetBookmarks(context.Background(), c.User, "", nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.Path, []byte(bookmarks), 0644); err != nil {
		return err
	}

	fmt.Printf("bookmarks exported to %s\n", c.Path)
	return nil
}
