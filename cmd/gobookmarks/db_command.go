package main

import (
	"flag"
	"fmt"
)

type DbCommand struct {
	parent Command
	Flags  *flag.FlagSet

	UsersCommand         *DbUsersCommand
	ResetPasswordCommand *DbResetPasswordCommand
	HelpCmd              *HelpCommand
}

func (rc *RootCommand) NewDbCommand() (*DbCommand, error) {
	c := &DbCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("db", flag.ContinueOnError),
	}
	c.UsersCommand, _ = c.NewDbUsersCommand()
	c.ResetPasswordCommand, _ = c.NewDbResetPasswordCommand()
	c.HelpCmd = NewHelpCommand(c)
	return c, nil
}

func (c *DbCommand) Name() string {
	return c.Flags.Name()
}

func (c *DbCommand) Parent() Command {
	return c.parent
}

func (c *DbCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *DbCommand) Subcommands() []Command {
	return []Command{c.UsersCommand, c.ResetPasswordCommand, c.HelpCmd}
}

func (c *DbCommand) Description() string {
	return "Inspect and modify database state"
}

func (c *DbCommand) Execute(args []string) error {
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
	case c.UsersCommand.Name():
		return c.UsersCommand.Execute(remaining[1:])
	case c.ResetPasswordCommand.Name():
		return c.ResetPasswordCommand.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown db subcommand: %s", remaining[0])
		printHelp(c, err)
		return err
	}
}
