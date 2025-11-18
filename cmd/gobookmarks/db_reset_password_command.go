package main

import (
	"context"
	"flag"
	"fmt"

	. "github.com/arran4/gobookmarks"
)

type DbResetPasswordCommand struct {
	parent Command
	Flags  *flag.FlagSet

	User     string
	Password string
}

func (dc *DbCommand) NewDbResetPasswordCommand() (*DbResetPasswordCommand, error) {
	c := &DbResetPasswordCommand{
		parent: dc,
		Flags:  flag.NewFlagSet("reset-password", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.User, "user", "", "username to reset password for")
	c.Flags.StringVar(&c.Password, "password", "", "new password")
	return c, nil
}

func (c *DbResetPasswordCommand) Name() string {
	return c.Flags.Name()
}

func (c *DbResetPasswordCommand) Parent() Command {
	return c.parent
}

func (c *DbResetPasswordCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *DbResetPasswordCommand) Subcommands() []Command {
	return nil
}

func (c *DbResetPasswordCommand) Description() string {
	return "Reset a user password inside the configured database"
}

func (c *DbResetPasswordCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if c.User == "" || c.Password == "" {
		err := fmt.Errorf("user and password are required")
		printHelp(c, err)
		return err
	}
	if len(c.Password) < 8 {
		err := fmt.Errorf("password must be at least 8 characters")
		printHelp(c, err)
		return err
	}

	cfg := c.Parent().(*DbCommand).parent.(*RootCommand).cfg

	if cfg.DBConnectionProvider == "" || cfg.DBConnectionString == "" {
		err := fmt.Errorf("database connection not configured")
		printHelp(c, err)
		return err
	}

	DBConnectionProvider = cfg.DBConnectionProvider
	DBConnectionString = cfg.DBConnectionString

	p := SQLProvider{}
	if err := p.SetPassword(context.Background(), c.User, c.Password); err != nil {
		printHelp(c, err)
		return err
	}

	fmt.Printf("password for user %s has been reset\n", c.User)
	return nil
}
