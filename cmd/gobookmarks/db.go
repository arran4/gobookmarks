package main

import (
	"context"
	"fmt"
	"flag"

	"github.com/arran4/gobookmarks"
)

func NewDbCmd() *DbCommand {
	return &DbCommand{
		FlagSet: flag.NewFlagSet("db", flag.ExitOnError),
	}
}

func (c *DbCommand) Run(args []string) error {
	if len(args) < 1 {
		c.Usage()
		return nil
	}

	switch args[0] {
	case "create-user":
		if len(args) != 3 {
			return fmt.Errorf("create-user requires exactly two arguments: the username and the password")
		}
		username := args[1]
		password := args[2]
		provider := gobookmarks.GetProvider("sql")
		if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
			return ph.CreateUser(context.Background(), username, password)
		}
		return fmt.Errorf("the sql provider does not support creating users")
	case "reset-password":
		if len(args) != 3 {
			return fmt.Errorf("reset-password requires exactly two arguments: the username and the new password")
		}
		username := args[1]
		password := args[2]
		provider := gobookmarks.GetProvider("sql")
		if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
			return ph.SetPassword(context.Background(), username, password)
		}
		return fmt.Errorf("the sql provider does not support resetting passwords")
	default:
		c.Usage()
		return nil
	}
}

func (c *DbCommand) Usage() {
	fmt.Println("Usage: gobookmarks db [command]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  create-user <username> <password>    Create a new user in the database")
	fmt.Println("  reset-password <username> <password> Reset a user's password")
}
