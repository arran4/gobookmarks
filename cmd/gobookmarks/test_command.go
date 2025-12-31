package main

import (
	"flag"
	"fmt"
)

type TestCommand struct {
	parent Command
	Flags  *flag.FlagSet

	VerificationCommand *VerificationCommand
	HelpCmd             *HelpCommand
}

func (rc *RootCommand) NewTestCommand() (*TestCommand, error) {
	c := &TestCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("test", flag.ContinueOnError),
	}
	c.VerificationCommand, _ = c.NewVerificationCommand()
	c.HelpCmd = NewHelpCommand(c)
	return c, nil
}

func (c *TestCommand) Name() string {
	return c.Flags.Name()
}

func (c *TestCommand) Parent() Command {
	return c.parent
}

func (c *TestCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *TestCommand) Subcommands() []Command {
	return []Command{c.VerificationCommand, c.HelpCmd}
}

func (c *TestCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		printHelp(c, nil)
		return nil
	}
	switch remaining[0] {
	case "-h", "--help", "help":
		return c.HelpCmd.Execute(remaining[1:])
	case c.VerificationCommand.Name():
		return c.VerificationCommand.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown test subcommand: %s", remaining[0])
		printHelp(c, err)
		return err
	}
}
