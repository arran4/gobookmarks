package main

import (
	"log"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &Command{
		Name:  "gobookmarks",
		Short: "A bookmark manager",
		Long:  `gobookmarks is a self-hosted bookmark manager written in Go.`,
	}

	rootCmd.AddCommand(NewServeCmd(version, commit, date))
	rootCmd.AddCommand(NewVersionCmd(version, commit, date))
	rootCmd.AddCommand(NewVerifyFileCmd())
	rootCmd.AddCommand(NewVerifyCredsCmd())
	rootCmd.AddCommand(NewDbCmd())
	rootCmd.AddCommand(NewGitCmd())

	if err := rootCmd.Execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
