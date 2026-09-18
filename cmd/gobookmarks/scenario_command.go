package main

import (
	"flag"
	"fmt"
)

type ScenarioCommand struct {
	parent Command
	Flags  *flag.FlagSet

	ValidateCommand *ScenarioValidateCommand
	ApplyCommand    *ScenarioApplyCommand
	ServeCommand    *ScenarioServeCommand
	HelpCmd         *HelpCommand
}

func (rc *RootCommand) NewScenarioCommand() (*ScenarioCommand, error) {
	c := &ScenarioCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("scenario", flag.ContinueOnError),
	}

	c.ValidateCommand, _ = c.NewScenarioValidateCommand()
	c.ApplyCommand, _ = c.NewScenarioApplyCommand()
	c.ServeCommand, _ = c.NewScenarioServeCommand()
	c.HelpCmd = NewHelpCommand(c)

	return c, nil
}

func (c *ScenarioCommand) Name() string {
	return c.Flags.Name()
}

func (c *ScenarioCommand) Parent() Command {
	return c.parent
}

func (c *ScenarioCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ScenarioCommand) Subcommands() []Command {
	return []Command{c.ValidateCommand, c.ApplyCommand, c.ServeCommand, c.HelpCmd}
}

func (c *ScenarioCommand) Execute(args []string) error {
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
	case c.ValidateCommand.Name():
		return c.ValidateCommand.Execute(remaining[1:])
	case c.ApplyCommand.Name():
		return c.ApplyCommand.Execute(remaining[1:])
	case c.ServeCommand.Name():
		return c.ServeCommand.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown scenario subcommand: %s", remaining[0])
		printHelp(c, err)
		return err
	}
}
