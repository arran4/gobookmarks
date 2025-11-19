package main

import (
	"flag"
	"fmt"
	"os"

	. "github.com/arran4/gobookmarks"
)

type VerifyFileCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Path   string
}

func (rc *RootCommand) NewVerifyFileCommand() (*VerifyFileCommand, error) {
	c := &VerifyFileCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("verify-file", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Path, "path", "", "path to the file to verify")
	return c, nil
}

func (c *VerifyFileCommand) Name() string {
	return c.Flags.Name()
}

func (c *VerifyFileCommand) Parent() Command {
	return c.parent
}

func (c *VerifyFileCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *VerifyFileCommand) Subcommands() []Command {
	return nil
}

func (c *VerifyFileCommand) Description() string {
	return "Validate a bookmarks file on disk"
}

func (c *VerifyFileCommand) Execute(args []string) error {
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

	data, err := os.ReadFile(c.Path)
	if err != nil {
		printHelp(c, err)
		return err
	}

	if _, err := ValidateBookmarks(string(data)); err != nil {
		printHelp(c, err)
		return err
	}

	fmt.Printf("%s is valid\n", c.Path)
	return nil
}
