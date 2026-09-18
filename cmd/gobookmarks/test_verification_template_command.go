package main

import (
	"flag"
	"fmt"
)

type TemplateCommand struct {
	parent Command
	Flags  *flag.FlagSet

	DataFromJSONFile stringFlag
	Serve            stringFlag
	Out              stringFlag
	HelpCmd          *HelpCommand
}

func (c *VerificationCommand) NewTemplateCommand() (*TemplateCommand, error) {
	tc := &TemplateCommand{
		parent: c,
		Flags:  flag.NewFlagSet("template", flag.ContinueOnError),
	}
	tc.Flags.Var(&tc.DataFromJSONFile, "data-from-json-file", "Path to JSON file containing template data")
	tc.Flags.Var(&tc.Serve, "serve", "Address to serve the template on (e.g. :8080)")
	tc.Flags.Var(&tc.Out, "out", "File to write output to")
	tc.HelpCmd = NewHelpCommand(tc)
	return tc, nil
}

func (c *TemplateCommand) Name() string {
	return c.Flags.Name()
}

func (c *TemplateCommand) Parent() Command {
	return c.parent
}

func (c *TemplateCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *TemplateCommand) Subcommands() []Command {
	return []Command{c.HelpCmd}
}

func (c *TemplateCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		return c.HelpCmd.Execute(nil)
	}

	subcommandRange := remaining[0]
	// Handle help within template command if user types "gobookmarks test verification template help"
	if subcommandRange == "help" || subcommandRange == "-h" || subcommandRange == "--help" {
		return c.HelpCmd.Execute(remaining[1:])
	}

	fmt.Printf("DEPRECATION WARNING: `test verification template` is deprecated.\n" +
		"Please use `scenario serve` for executable application state scenarios.\n" +
		"Migrating to `scenario serve` under the hood...\n")

	scenarioFile := ""
	switch subcommandRange {
	case "complex":
		scenarioFile = "scenarios/complex-bookmarks.txtar"
	default:
		// Default to complex as it covers most UI elements
		scenarioFile = "scenarios/complex-bookmarks.txtar"
	}

	rc := c.parent.Parent().(*RootCommand)
	serveCmd, err := rc.ScenarioCmd.NewScenarioServeCommand()
	if err != nil {
		return fmt.Errorf("failed to instantiate scenario serve command: %w", err)
	}

	serveArgs := []string{scenarioFile}
	if c.Serve.set {
		serveArgs = append(serveArgs, "--port", c.Serve.value)
	}

	return serveCmd.Execute(serveArgs)
}
