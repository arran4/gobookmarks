package main

import (
	"flag"
	"fmt"
)

type VerificationCommand struct {
	parent Command
	Flags  *flag.FlagSet

	TemplateCommand *TemplateCommand
	HelpCmd         *HelpCommand
}

func (c *TestCommand) NewVerificationCommand() (*VerificationCommand, error) {
	vc := &VerificationCommand{
		parent: c,
		Flags:  flag.NewFlagSet("verification", flag.ContinueOnError),
	}
	vc.TemplateCommand, _ = vc.NewTemplateCommand()
	vc.HelpCmd = NewHelpCommand(vc)
	return vc, nil
}

func (c *VerificationCommand) Name() string {
	return c.Flags.Name()
}

func (c *VerificationCommand) Parent() Command {
	return c.parent
}

func (c *VerificationCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *VerificationCommand) Subcommands() []Command {
	return []Command{c.TemplateCommand, c.HelpCmd}
}

func (c *VerificationCommand) Execute(args []string) error {
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
	case c.TemplateCommand.Name():
		return c.TemplateCommand.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown verification subcommand: %s", remaining[0])
		printHelp(c, err)
		return err
	}
}
