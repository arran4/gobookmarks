package main

import (
	"flag"
	"fmt"
	"os"
)

type ScenarioValidateCommand struct {
	parent Command
	Flags  *flag.FlagSet
}

func (sc *ScenarioCommand) NewScenarioValidateCommand() (*ScenarioValidateCommand, error) {
	c := &ScenarioValidateCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("validate", flag.ContinueOnError),
	}
	return c, nil
}

func (c *ScenarioValidateCommand) Name() string {
	return c.Flags.Name()
}

func (c *ScenarioValidateCommand) Parent() Command {
	return c.parent
}

func (c *ScenarioValidateCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ScenarioValidateCommand) Subcommands() []Command {
	return nil
}

func (c *ScenarioValidateCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		return fmt.Errorf("missing scenario PATH")
	}

	path := remaining[0]
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open scenario: %w", err)
	}
	defer func() { _ = file.Close() }()

	scenario, err := ParseScenario(file)
	if err != nil {
		return fmt.Errorf("failed to parse scenario: %w", err)
	}

	if err := ValidateScenario(scenario); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("Scenario validation successful for %s\n", path)
	return nil
}
