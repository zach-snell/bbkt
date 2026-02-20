package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "Manage Bitbucket workspaces",
}

var workspacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces the authenticated user has access to",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListWorkspaces(bitbucket.ListWorkspacesArgs{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

var workspacesGetCmd = &cobra.Command{
	Use:   "get [workspace]",
	Short: "Get details for a specific workspace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.GetWorkspace(bitbucket.GetWorkspaceArgs{
			Workspace: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

func init() {
	RootCmd.AddCommand(workspacesCmd)
	workspacesCmd.AddCommand(workspacesListCmd)
	workspacesCmd.AddCommand(workspacesGetCmd)
}

// getClient is a helper to instantiate the core Bitbucket API client
func getClient() *bitbucket.Client {
	username := os.Getenv("BITBUCKET_USERNAME")
	password := os.Getenv("BITBUCKET_API_TOKEN")
	token := os.Getenv("BITBUCKET_ACCESS_TOKEN")

	if token != "" || (username != "" && password != "") {
		return bitbucket.NewClient(username, password, token)
	}

	creds, err := bitbucket.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not authenticated. Run 'bbkt auth' first.\n")
		os.Exit(1)
	}

	var c *bitbucket.Client
	if creds.IsAPIToken() {
		c = bitbucket.NewClient(creds.Email, creds.APIToken, "")
	} else if creds.IsOAuth() {
		// Auto refresh if needed
		if creds.IsExpired() {
			err = bitbucket.RefreshOAuth(creds)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to refresh oauth token. Run 'bbkt auth' again.\n")
				os.Exit(1)
			}
		}
		c = bitbucket.NewClient("", "", creds.AccessToken)
	}

	return c
}
