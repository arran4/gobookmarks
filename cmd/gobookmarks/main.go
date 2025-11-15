package main

import (
	"log"
	"os"

	"github.com/arran4/gobookmarks/cmd/gobookmarks/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cli.Command{
		Name:  "gobookmarks",
		Short: "A bookmark manager",
		Long:  `gobookmarks is a self-hosted bookmark manager written in Go.`,
	}

	rootCmd.AddCommand(cli.NewServeCmd(version, commit, date))
	rootCmd.AddCommand(cli.NewVersionCmd(version, commit, date))
	rootCmd.AddCommand(cli.NewVerifyFileCmd())
	rootCmd.AddCommand(cli.NewVerifyCredsCmd())
	rootCmd.AddCommand(cli.NewDbCmd())
	rootCmd.AddCommand(cli.NewGitCmd())

	if err := rootCmd.Execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
