package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var (
	useOAuth    bool
	profileName string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Bitbucket Cloud",
	Long: `Set up credentials for accessing Bitbucket Cloud.

By default, this sets up an API Token (Basic Auth).
If you prefer an OAuth 2.0 flow (requires workspace admin), use the --oauth flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		if useOAuth {
			runOAuthLogin(profileName)
			return
		}
		if err := bitbucket.APITokenLogin(profileName); err != nil {
			fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	Run: func(cmd *cobra.Command, args []string) {
		runLogout()
	},
}

func init() {
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(logoutCmd)

	authCmd.Flags().BoolVar(&useOAuth, "oauth", false, "Authenticate via OAuth (opens browser)")
	authCmd.Flags().StringVarP(&profileName, "profile", "p", "default", "Profile name to save these credentials under")
}

func runOAuthLogin(profile string) {
	clientID := os.Getenv("BITBUCKET_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("BITBUCKET_OAUTH_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Fprintf(os.Stderr, "OAuth credentials required. Set:\n")
		fmt.Fprintf(os.Stderr, "  BITBUCKET_OAUTH_CLIENT_ID\n")
		fmt.Fprintf(os.Stderr, "  BITBUCKET_OAUTH_CLIENT_SECRET\n\n")
		fmt.Fprintf(os.Stderr, "Create an OAuth consumer at:\n")
		fmt.Fprintf(os.Stderr, "  Bitbucket > Workspace Settings > OAuth consumers > Add consumer\n")
		fmt.Fprintf(os.Stderr, "  Callback URL: http://localhost:<any-port>/callback\n")
		fmt.Fprintf(os.Stderr, "  Scopes: repository, repository:write, pullrequest, pullrequest:write,\n")
		fmt.Fprintf(os.Stderr, "          pipeline, pipeline:write, account\n")
		os.Exit(1)
	}

	if err := bitbucket.OAuthLogin(clientID, clientSecret, profile); err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		os.Exit(1)
	}
}

func runStatus() {
	creds, err := bitbucket.LoadCredentials()
	if err != nil {
		if os.Getenv("BITBUCKET_ACCESS_TOKEN") != "" {
			fmt.Println("Authenticated via BITBUCKET_ACCESS_TOKEN environment variable")
			return
		}
		if os.Getenv("BITBUCKET_USERNAME") != "" && os.Getenv("BITBUCKET_API_TOKEN") != "" {
			fmt.Println("Authenticated via BITBUCKET_USERNAME + BITBUCKET_API_TOKEN environment variables")
			return
		}
		fmt.Println("Not authenticated. Run: bbkt auth")
		return
	}

	path, _ := bitbucket.CredentialsPath()

	switch {
	case creds.IsAPIToken():
		fmt.Println("Authenticated via API Token (Basic Auth)")
		fmt.Printf("  Profile: %s\n", creds.ProfileName)
		fmt.Printf("  Email:   %s\n", creds.Email)
		if len(creds.APIToken) > 8 {
			fmt.Printf("  Token:   %s...%s\n", creds.APIToken[:4], creds.APIToken[len(creds.APIToken)-4:])
		} else {
			fmt.Println("  Token:   ****")
		}
		if creds.Scopes != "" {
			fmt.Printf("  Scopes:  %s\n", creds.Scopes)
		}
		fmt.Printf("  Stored:  %s\n", creds.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  File:    %s\n", path)

	case creds.IsOAuth():
		fmt.Println("Authenticated via OAuth 2.0 (Bearer Auth)")
		fmt.Printf("  Scopes:  %s\n", creds.Scopes)
		fmt.Printf("  Stored:  %s\n", creds.CreatedAt.Format("2006-01-02 15:04:05"))
		if creds.IsExpired() {
			fmt.Println("  Status:  expired (will auto-refresh)")
		} else {
			fmt.Println("  Status:  valid")
		}
		fmt.Printf("  File:    %s\n", path)
	}
}

func runLogout() {
	if err := bitbucket.RemoveCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "error removing credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged out. Credentials removed.")
}
