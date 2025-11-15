package main

import (
	"context"
	"fmt"

	"github.com/arran4/gobookmarks"
)

func (c *GitCommand) Name() string  { return "git" }
func (c *GitCommand) Short() string { return "Git storage inspection commands" }
func (c *GitCommand) Long() string  { return `Commands for inspecting and managing the git storage.` }
func (c *GitCommand) Run(args []string) error {
	return c.Usage()
}

type GitCreateUserCommand struct {
	*GitCommand
}

func (c *GitCommand) AddCreateUserCmd() {
	cmd := &GitCreateUserCommand{GitCommand: c}
	c.AddCommand(cmd)
}

func (c *GitCreateUserCommand) Name() string  { return "create-user" }
func (c *GitCreateUserCommand) Short() string { return "Create a new user in the git storage" }
func (c *GitCreateUserCommand) Long() string  { return `Creates a new user in the git storage.` }
func (c *GitCreateUserCommand) Run(args []string) error {
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
}
