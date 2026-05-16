package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var workspacesCmd = &cobra.Command{
	Use:     "workspaces",
	GroupID: groupData,
	Short:   "List and inspect Bitbucket workspaces you have access to",
	Long:    "Read-only operations on the workspaces your authenticated user (or token) has access to.",
	Example: `  bbkt workspaces list             # all workspaces
  bbkt workspaces get my-team`,
}

var workspacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces the authenticated user has access to",
	RunE: func(cmd *cobra.Command, args []string) error {
		page, pagelen := paginationArgs(cmd)
		client := getClient()
		result, err := client.ListWorkspaces(bitbucket.ListWorkspacesArgs{
			Page:    page,
			Pagelen: pagelen,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No workspaces found.")
				return
			}
			t := NewTable()
			t.Header("Name", "Slug", "Visibility")
			for _, w := range result.Values {
				t.Row(w.Name, w.Slug, FormatPrivate(w.IsPrivate))
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
		return nil
	},
}

var workspacesGetCmd = &cobra.Command{
	Use:   "get <workspace>",
	Short: "Get details for a specific workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getClient()
		result, err := client.GetWorkspace(bitbucket.GetWorkspaceArgs{
			Workspace: args[0],
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Workspace: %s\n", result.Name)
			KV("Slug", result.Slug)
			KV("UUID", result.UUID)
			KV("Visibility", FormatPrivate(result.IsPrivate))
		})
		return nil
	},
}

func init() {
	RootCmd.AddCommand(workspacesCmd)
	workspacesCmd.AddCommand(workspacesListCmd)
	workspacesCmd.AddCommand(workspacesGetCmd)
	addPaginationFlags(workspacesListCmd)
}

// getClient is a helper to instantiate the core Bitbucket API client.
// Exits if no credentials are configured — this is a setup error, not a
// runtime API error, and the actionable message is more useful than
// propagating it through RunE.
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
