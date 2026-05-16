package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage credential profiles (list, switch, refresh)",
	Long: `Manage Bitbucket credential profiles for multiple accounts (e.g.
personal vs work). Each profile is a separate set of credentials
stored in ~/.config/bbkt/credentials.json.

Running 'bbkt profile' with no subcommand lists profiles. Use
'bbkt profile use <name>' to change the active profile, or pass
--profile <name> (or set BBKT_PROFILE) to override per-command.`,
	Example: `  bbkt profile                     # list profiles
  bbkt profile use work            # set active profile
  bbkt profile refresh             # refresh cached workspace list
  bbkt --profile work prs list     # one-shot override`,
	Run: func(cmd *cobra.Command, args []string) {
		listProfiles()
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured credential profiles",
	Run: func(cmd *cobra.Command, args []string) {
		listProfiles()
	},
}

func listProfiles() {
	store, err := bitbucket.LoadProfileStore()
	if err != nil {
		fmt.Println("No profiles found. Run 'bbkt auth' to create one.")
		return
	}

	// Also check what Local Git says so we can indicate auto-detection
	localWorkspace, _, werr := bitbucket.GetLocalRepoInfo()

	fmt.Println("Available Profiles:")
	for name, cred := range store.Profiles {
		active := ""
		if name == store.ActiveProfile {
			active = " (active)"
		}
		if werr == nil && localWorkspace != "" {
			for _, ws := range cred.AccessibleWorkspaces {
				if ws == localWorkspace {
					active += " [matches current git workspace]"
					break
				}
			}
		}
		fmt.Printf("  - %s: %s%s\n", name, cred.Email, active)
	}
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default active profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		store, err := bitbucket.LoadProfileStore()
		if err != nil {
			fmt.Println("Error loading profiles.")
			return
		}

		if _, ok := store.Profiles[name]; !ok {
			fmt.Printf("Profile '%s' not found.\n", name)
			os.Exit(1)
		}

		store.ActiveProfile = name
		if err := bitbucket.SaveProfileStore(store); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving profile: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Default profile set to '%s'\n", name)
	},
}

var profileRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh workspace cache for all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := bitbucket.LoadProfileStore()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading profiles.")
			return
		}

		fmt.Println("Refreshing workspaces...")
		for name, cred := range store.Profiles {
			var client *bitbucket.Client
			if cred.IsAPIToken() {
				client = bitbucket.NewClient(cred.Email, cred.APIToken, "")
			} else if cred.IsOAuth() {
				if cred.IsExpired() {
					_ = bitbucket.RefreshOAuth(cred)
				}
				client = bitbucket.NewClient("", "", cred.AccessToken)
			}
			if client != nil {
				slugs := bitbucket.FetchAccessibleWorkspaces(client)
				cred.AccessibleWorkspaces = slugs
				fmt.Printf("  - %s: Found %d workspaces\n", name, len(slugs))
			}
		}

		if err := bitbucket.SaveProfileStore(store); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving profiles: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done.")
	},
}

func init() {
	RootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileRefreshCmd)
}
