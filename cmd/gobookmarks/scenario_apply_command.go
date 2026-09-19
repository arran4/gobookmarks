package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

type ScenarioApplyCommand struct {
	parent Command
	Flags  *flag.FlagSet
}

func (sc *ScenarioCommand) NewScenarioApplyCommand() (*ScenarioApplyCommand, error) {
	c := &ScenarioApplyCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("apply", flag.ContinueOnError),
	}
	return c, nil
}

func (c *ScenarioApplyCommand) Name() string {
	return c.Flags.Name()
}

func (c *ScenarioApplyCommand) Parent() Command {
	return c.parent
}

func (c *ScenarioApplyCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ScenarioApplyCommand) Subcommands() []Command {
	return nil
}

func (c *ScenarioApplyCommand) Execute(args []string) error {
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

	// `scenario apply` is a validation/seeding rehearsal. It always creates a
	// disposable runtime and removes it before returning; it never mutates a
	// configured backend.
	cleanup, err := setupScenarioBackend()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if err := ApplyScenario(context.Background(), scenario); err != nil {
		return fmt.Errorf("failed to apply scenario: %w", err)
	}

	fmt.Printf("Scenario applied successfully to disposable state for %s (state removed on exit)\n", path)
	return nil
}
