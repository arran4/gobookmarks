package main

import (
	"context"
	"fmt"

	"github.com/arran4/gobookmarks"
)

func NewDbCmd() *Command {
	dbCmd := &Command{
		Name:  "db",
		Short: "Database inspection commands",
		Long:  `Commands for inspecting and managing the database.`,
	}

	createUserCmd := &Command{
		Name:  "create-user",
		Short: "Create a new user in the database",
		Long:  `Creates a new user in the database.`,
		Run: func(cmd *Command, args []string) error {
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
		},
	}

	resetPasswordCmd := &Command{
		Name:  "reset-password",
		Short: "Reset a user's password",
		Long:  `Resets a user's password in the database.`,
		Run: func(cmd *Command, args []string) error {
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
		},
	}

	dbCmd.AddCommand(createUserCmd)
	dbCmd.AddCommand(resetPasswordCmd)
	return dbCmd
}
