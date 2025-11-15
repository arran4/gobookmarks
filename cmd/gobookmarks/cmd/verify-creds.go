package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var verifyCredsCmd = &cobra.Command{
	Use:   "verify-creds [provider]",
	Short: "Verify OAuth2 credentials",
	Long:  `Checks the validity of OAuth2 credentials for a given provider (e.g., github, gitlab).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		fmt.Printf("Verifying credentials for provider: %s\n", provider)
		// TODO: Implement credential verification logic
	},
}

func init() {
	rootCmd.AddCommand(verifyCredsCmd)
}
