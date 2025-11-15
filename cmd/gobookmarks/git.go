package main

import (
	"context"
	"fmt"
	"flag"

	"github.com/arran4/gobookmarks"
)

func NewGitCmd() *GitCommand {
	return &GitCommand{
		FlagSet: flag.NewFlagSet("git", flag.ExitOnError),
	}
}

func (c *GitCommand) Run(args []string) error {
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
		provider := gobookmarks.GetProvider("git")
		if ph, ok := provider.(gobookmarks.PasswordHandler); ok {
			return ph.CreateUser(context.Background(), username, password)
		}
		return fmt.Errorf("the git provider does not support creating users")
	default:
		c.Usage()
		return nil
	}
}

func (c *GitCommand) Usage() {
	fmt.Println("Usage: gobookmarks git [command]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  create-user <username> <password>    Create a new user in the git storage")
}
