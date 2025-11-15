package main

import (
	"context"
	"fmt"

	"github.com/arran4/gobookmarks"
)

func NewGitCmd() *Command {
	gitCmd := &Command{
		Name:  "git",
		Short: "Git storage inspection commands",
		Long:  `Commands for inspecting and managing the git storage.`,
	}

	createUserCmd := &Command{
		Name:  "create-user",
		Short: "Create a new user in the git storage",
		Long:  `Creates a new user in the git storage.`,
		Run: func(cmd *Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("create-user requires exactly two arguments: the username and the password")
			}
			username := args[0]
			password := args[1]
			provider := gobookmarks.GetProvider("git")
			if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
				return ph.CreateUser(context.Background(), username, password)
			}
			return fmt.Errorf("the git provider does not support creating users")
		},
	}
	gitCmd.AddCommand(createUserCmd)
	return gitCmd
}
