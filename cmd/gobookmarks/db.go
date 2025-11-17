package main

import (
	"flag"
	"fmt"
)

type DbCommand struct {
	*RootCommand
	fs *flag.FlagSet
}

func NewDbCommand(root *RootCommand) *DbCommand {
	c := &DbCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("db", flag.ExitOnError),
	}
	return c
}

func (c *DbCommand) Name() string {
	return c.fs.Name()
}

func (c *DbCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *DbCommand) Execute(args []string) error {
	c.fs.Usage = func() { printHelp(c, c.subcommands()...) }
	if len(args) < 1 {
		c.fs.Usage()
		return nil
	}

	var cmd Command
	subcommands := c.subcommands()
	for _, sub := range subcommands {
		if sub.Name() == args[0] {
			cmd = sub
			break
		}
	}

	if cmd == nil {
		if args[0] == "help" {
			if len(args) < 2 {
				c.fs.Usage()
				return nil
			}
			for _, sub := range subcommands {
				if sub.Name() == args[1] {
					sub.Fs().Usage()
					return nil
				}
			}
			return fmt.Errorf("unknown db subcommand: %s", args[1])
		}
		return fmt.Errorf("unknown db subcommand: %s", args[0])
	}

	cmd.Fs().Usage = func() { printHelp(cmd) }
	for _, arg := range args[1:] {
		if arg == "-h" || arg == "-help" {
			cmd.Fs().Usage()
			return nil
		}
	}

	return cmd.Execute(args[1:])
}

func (c *DbCommand) subcommands() []Command {
	return []Command{
		NewDbUsersCommand(c.RootCommand),
		NewDbResetPasswordCommand(c.RootCommand),
	}
}
