package main

import "flag"

type HelpCommand struct {
	parent Command
	Flags  *flag.FlagSet
}

func NewHelpCommand(parent Command) *HelpCommand {
	return &HelpCommand{
		parent: parent,
		Flags:  flag.NewFlagSet("help", flag.ContinueOnError),
	}
}

func (c *HelpCommand) Name() string {
	return c.Flags.Name()
}

func (c *HelpCommand) Parent() Command {
	return c.parent
}

func (c *HelpCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *HelpCommand) Subcommands() []Command {
	return nil
}

func (c *HelpCommand) Description() string {
	return "Display contextual help for a command"
}

func (c *HelpCommand) Execute(args []string) error {
	target := c.parent
	if len(args) > 0 {
		for _, sub := range c.parent.Subcommands() {
			if sub.Name() == args[0] {
				target = sub
				break
			}
		}
	}
	c.FlagSet().Parse(args)
	c.FlagSet().Usage = func() {}
	printHelp(target, nil)
	return nil
}
