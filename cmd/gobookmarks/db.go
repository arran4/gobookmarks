package main

import (
	"context"
	"fmt"

	"github.com/arran4/gobookmarks"
)

func (c *DbCommand) Name() string  { return "db" }
func (c *DbCommand) Short() string { return "Database inspection commands" }
func (c *DbCommand) Long() string  { return `Commands for inspecting and managing the database.` }
func (c *DbCommand) Run(args []string) error {
	return c.Usage()
}

type DbCreateUserCommand struct {
	*DbCommand
}

func (c *DbCommand) AddCreateUserCmd() {
	cmd := &DbCreateUserCommand{DbCommand: c}
	c.AddCommand(cmd)
}

func (c *DbCreateUserCommand) Name() string  { return "create-user" }
func (c *DbCreateUserCommand) Short() string { return "Create a new user in the database" }
func (c *DbCreateUserCommand) Long() string  { return `Creates a new user in the database.` }
func (c *DbCreateUserCommand) Run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("create-user requires exactly two arguments: the username and the password")
	}
	username := args[0]
	password := args[1]
	provider := gobookmarks.GetProvider("sql")
	if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
		return ph.CreateUser(context.Background(), username, password)
	}
	return fmt.Errorf("the sql provider does not support creating users")
}

type DbResetPasswordCommand struct {
	*DbCommand
}

func (c *DbCommand) AddResetPasswordCmd() {
	cmd := &DbResetPasswordCommand{DbCommand: c}
	c.AddCommand(cmd)
}

func (c *DbResetPasswordCommand) Name() string  { return "reset-password" }
func (c *DbResetPasswordCommand) Short() string { return "Reset a user's password" }
func (c *DbResetPasswordCommand) Long() string  { return `Resets a user's password in the database.` }
func (c *DbResetPasswordCommand) Run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("reset-password requires exactly two arguments: the username and the new password")
	}
	username := args[0]
	password := args[1]
	provider := gobookmarks.GetProvider("sql")
	if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
		return ph.SetPassword(context.Background(), username, password)
	}
	return fmt.Errorf("the sql provider does not support resetting passwords")
}
