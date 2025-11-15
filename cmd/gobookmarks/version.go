package main

import (
	"fmt"
	"strings"

	"github.com/arran4/gobookmarks"
)

func NewVersionCmd(version, commit, date string) *Command {
	return &Command{
		Name:  "version",
		Short: "Print the version number of gobookmarks",
		Long:  `All software has versions. This is gobookmarks'`,
		Run: func(cmd *Command, args []string) error {
			fmt.Printf("gobookmarks %s commit %s built %s\n", version, commit, date)
			fmt.Printf("providers: %s\n", strings.Join(gobookmarks.ProviderNames(), ", "))
			return nil
		},
	}
}
