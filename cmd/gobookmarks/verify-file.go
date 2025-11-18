package main

import (
	"flag"
	"fmt"
	"os"

	. "github.com/arran4/gobookmarks"
)

type VerifyFileCommand struct {
	*RootCommand
	fs   *flag.FlagSet
	Path string
}

func NewVerifyFileCommand(root *RootCommand) *VerifyFileCommand {
	c := &VerifyFileCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("verify-file", flag.ExitOnError),
	}
	c.fs.StringVar(&c.Path, "path", "", "Path to the bookmarks file")
	return c
}

func (c *VerifyFileCommand) Name() string {
	return c.fs.Name()
}

func (c *VerifyFileCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *VerifyFileCommand) Execute(args []string) error {
	c.fs.Parse(args)
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	fmt.Printf("verifying file %s...\n", c.Path)

	b, err := os.ReadFile(c.Path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	bookmarks := ParseBookmarks(string(b))
	if len(bookmarks) == 0 {
		return fmt.Errorf("no bookmarks found in file")
	}

	fmt.Println("file verified successfully")
	return nil
}
