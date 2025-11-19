package main

import (
	"flag"
	"fmt"
)

type VerifyCredsCommand struct {
	parent Command
	Flags  *flag.FlagSet
	User   string
	Pass   string
}

func (rc *RootCommand) NewVerifyCredsCommand() (*VerifyCredsCommand, error) {
	c := &VerifyCredsCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("verify-creds", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.User, "user", "", "username to verify")
	c.Flags.StringVar(&c.Pass, "pass", "", "password to verify")
	return c, nil
}

func (c *VerifyCredsCommand) Name() string {
	return c.Flags.Name()
}

func (c *VerifyCredsCommand) Parent() Command {
	return c.parent
}

func (c *VerifyCredsCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *VerifyCredsCommand) Subcommands() []Command {
	return nil
}

func (c *VerifyCredsCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}
	if c.User == "" || c.Pass == "" {
		err := fmt.Errorf("user and pass are required")
		printHelp(c, err)
		return err
	}

	provider, err := getConfiguredProvider(&c.parent.(*RootCommand).cfg)
	if err != nil {
		printHelp(c, err)
		return err
	}

	ok, err := provider.VerifyCredentials(c.User, c.Pass)
	if err != nil {
		printHelp(c, err)
		return err
	}
	if !ok {
		err := fmt.Errorf("credentials rejected for %s", c.User)
		printHelp(c, err)
		return err
	}

	fmt.Printf("credentials valid for %s\n", c.User)
	return nil
}
