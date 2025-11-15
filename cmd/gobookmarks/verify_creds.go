package main

import (
	"fmt"
	"os"
)

func NewVerifyCredsCmd() *Command {
	var (
		ghID     string
		ghSecret string
		glID     string
		glSecret string
	)

	cmd := &Command{
		Name:  "verify-creds",
		Short: "Verify OAuth2 credentials",
		Long:  `Checks that OAuth2 credentials (client ID and secret) are present for a given provider.`,
		Run: func(cmd *Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("verify-creds requires exactly one argument: the provider (e.g., github, gitlab)")
			}
			provider := args[0]

			// Load from flags first, then fall back to environment variables
			if ghID == "" {
				ghID = os.Getenv("GITHUB_CLIENT_ID")
			}
			if ghSecret == "" {
				ghSecret = os.Getenv("GITHUB_SECRET")
			}
			if glID == "" {
				glID = os.Getenv("GITLAB_CLIENT_ID")
			}
			if glSecret == "" {
				glSecret = os.Getenv("GITLAB_SECRET")
			}

			switch provider {
			case "github":
				if ghID == "" || ghSecret == "" {
					return fmt.Errorf("github client ID or secret is missing. Set GITHUB_CLIENT_ID and GITHUB_SECRET environment variables, or use flags")
				}
				fmt.Println("GitHub credentials are present.")
			case "gitlab":
				if glID == "" || glSecret == "" {
					return fmt.Errorf("gitlab client ID or secret is missing. Set GITLAB_CLIENT_ID and GITLAB_SECRET environment variables, or use flags")
				}
				fmt.Println("GitLab credentials are present.")
			default:
				return fmt.Errorf("unknown provider: %s. Supported providers are github, gitlab", provider)
			}
			return nil
		},
	}

	cmd.FlagSet.StringVar(&ghID, "github-client-id", "", "GitHub OAuth client ID")
	cmd.FlagSet.StringVar(&ghSecret, "github-secret", "", "GitHub OAuth client secret")
	cmd.FlagSet.StringVar(&glID, "gitlab-client-id", "", "GitLab OAuth client ID")
	cmd.FlagSet.StringVar(&glSecret, "gitlab-secret", "", "GitLab OAuth client secret")

	return cmd
}
