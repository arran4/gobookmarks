package cmd

import (
	"fmt"
	"strings"

	"github.com/arran4/gobookmarks"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of gobookmarks",
	Long:  `All software has versions. This is gobookmarks'`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gobookmarks %s commit %s built %s\n", version, commit, date)
		fmt.Printf("providers: %s\n", strings.Join(gobookmarks.ProviderNames(), ", "))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
