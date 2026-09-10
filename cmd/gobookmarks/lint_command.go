package main

import (
	"flag"
	"fmt"
	"os"

	gobookmarks "github.com/arran4/gobookmarks"
)

type LintCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Path   string
}

func (rc *RootCommand) NewLintCommand() (*LintCommand, error) {
	c := &LintCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("lint", flag.ContinueOnError),
	}
	// Support alias 'verify-file' but canonical is 'lint'
	c.Flags.StringVar(&c.Path, "path", "", "path to the file to lint/verify (deprecated, prefer passing file as an argument)")
	return c, nil
}

func (c *LintCommand) Name() string {
	return c.Flags.Name()
}

func (c *LintCommand) Parent() Command {
	return c.parent
}

func (c *LintCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *LintCommand) Subcommands() []Command {
	return nil
}

func (c *LintCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	targetPath := c.Path
	if targetPath == "" {
		if c.FlagSet().NArg() > 0 {
			targetPath = c.FlagSet().Arg(0)
		} else {
			err := fmt.Errorf("file path is required as an argument or via -path")
			printHelp(c, err)
			return err
		}
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", targetPath, err)
	}

	if _, err := gobookmarks.StrictParseBookmarks(string(data)); err != nil {
		// Output the error directly for linters. It already includes the line number/context.
		return fmt.Errorf("lint error in %q: %w", targetPath, err)
	}

	fmt.Printf("%s is valid\n", targetPath)
	return nil
}
