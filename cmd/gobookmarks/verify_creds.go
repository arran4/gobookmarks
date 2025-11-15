package main

import (
	"fmt"
	"os"
)

func (c *VerifyCredsCommand) Name() string  { return "verify-creds" }
func (c *VerifyCredsCommand) Short() string { return "Verify OAuth2 credentials" }
func (c *VerifyCredsCommand) Long() string {
	return `Checks that OAuth2 credentials (client ID and secret) are present for a given provider.`
}

func (c *VerifyCredsCommand) Run(args []string) error {
	var (
		ghID     string
		ghSecret string
		glID     string
		glSecret string
	)

	fs := c.FlagSet()
	fs.StringVar(&ghID, "github-client-id", "", "GitHub OAuth client ID")
	fs.StringVar(&ghSecret, "github-secret", "", "GitHub OAuth client secret")
	fs.StringVar(&glID, "gitlab-client-id", "", "GitLab OAuth client ID")
	fs.StringVar(&glSecret, "gitlab-secret", "", "GitLab OAuth client secret")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) != 1 {
		return fmt.Errorf("verify-creds requires exactly one argument: the provider (e.g., github, gitlab)")
	}
	provider := fs.Arg(0)

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
}
