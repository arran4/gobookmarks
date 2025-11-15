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
	rootCmd := NewRootCommand(version, commit, date)
	rootCmd.AddServeCmd()
	rootCmd.AddVersionCmd()
	rootCmd.AddVerifyFileCmd()
	rootCmd.AddVerifyCredsCmd()

	dbCmd := &DbCommand{RootCommand: rootCmd}
	dbCmd.AddCreateUserCmd()
	dbCmd.AddResetPasswordCmd()
	rootCmd.AddCommand(dbCmd)

	gitCmd := &GitCommand{RootCommand: rootCmd}
	gitCmd.AddCreateUserCmd()
	rootCmd.AddCommand(gitCmd)

	if err := rootCmd.Execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
