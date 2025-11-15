package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git storage inspection commands",
	Long:  `Commands for inspecting and managing the git storage.`,
}

var gitListUsersCmd = &cobra.Command{
	Use:   "list-users",
	Short: "List all users in the git storage",
	Long:  `Lists all users in the git storage.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing users from the git storage...")
		// TODO: Implement user listing logic for git storage
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
	gitCmd.AddCommand(gitListUsersCmd)
}
