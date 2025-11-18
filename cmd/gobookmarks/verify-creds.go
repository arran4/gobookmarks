package main

import (
	"flag"
	"fmt"
)

type VerifyCredsCommand struct {
	*RootCommand
	fs *flag.FlagSet
}

func NewVerifyCredsCommand(root *RootCommand) *VerifyCredsCommand {
	c := &VerifyCredsCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("verify-creds", flag.ExitOnError),
	}
	return c
}

func (c *VerifyCredsCommand) Name() string {
	return c.fs.Name()
}

func (c *VerifyCredsCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *VerifyCredsCommand) Execute(args []string) error {
	c.fs.Parse(args)
	fmt.Println("verifying credentials...")
	cfg := c.RootCommand.cfg

	if cfg.GithubClientID != "" && cfg.GithubSecret != "" {
		fmt.Println("GitHub credentials found")
	} else {
		fmt.Println("GitHub credentials not found")
	}

	if cfg.GitlabClientID != "" && cfg.GitlabSecret != "" {
		fmt.Println("GitLab credentials found")
	} else {
		fmt.Println("GitLab credentials not found")
	}

	return nil
}
