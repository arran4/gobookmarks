package cmd

import (
	"fmt"
	"os"

	"github.com/arran4/gobookmarks/cmd/gobookmarks/cmd/templates"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gobookmarks",
	Short: "A bookmark manager",
	Long:  `gobookmarks is a self-hosted bookmark manager written in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to serve command
		serveCmd.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetHelpTemplate(templates.HelpTemplate)
}

func initConfig() {
	// Add any future configuration initialization here.
}
