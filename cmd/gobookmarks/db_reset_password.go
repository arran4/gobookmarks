package main

import (
	"context"
	"flag"
	"fmt"

	. "github.com/arran4/gobookmarks"
)

type DbResetPasswordCommand struct {
	*RootCommand
	fs *flag.FlagSet

	User     string
	Password string
}

func NewDbResetPasswordCommand(root *RootCommand) *DbResetPasswordCommand {
	c := &DbResetPasswordCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("reset-password", flag.ExitOnError),
	}
	c.fs.StringVar(&c.User, "user", "", "username to reset password for")
	c.fs.StringVar(&c.Password, "password", "", "new password")
	return c
}

func (c *DbResetPasswordCommand) Name() string {
	return c.fs.Name()
}

func (c *DbResetPasswordCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *DbResetPasswordCommand) Execute(args []string) error {
	c.fs.Parse(args)
	if c.User == "" || c.Password == "" {
		return fmt.Errorf("user and password are required")
	}
	if len(c.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	cfg := c.RootCommand.cfg

	if cfg.DBConnectionProvider == "" || cfg.DBConnectionString == "" {
		return fmt.Errorf("database connection not configured")
	}

	DBConnectionProvider = cfg.DBConnectionProvider
	DBConnectionString = cfg.DBConnectionString

	p := SQLProvider{}
	if err := p.SetPassword(context.Background(), c.User, c.Password); err != nil {
		return err
	}

	fmt.Printf("password for user %s has been reset\n", c.User)
	return nil
}
