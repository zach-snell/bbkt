package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var profileCmd = &cobra.Command{
	Use:     "profile",
	GroupID: groupAuth,
	Short:   "Manage credential profiles (list, switch, refresh)",
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
	RunE: func(cmd *cobra.Command, args []string) error {
		listProfiles()
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured credential profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		listProfiles()
		return nil
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
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		store, err := bitbucket.LoadProfileStore()
		if err != nil {
			return fmt.Errorf("loading profiles: %w", err)
		}

		if _, ok := store.Profiles[name]; !ok {
			return fmt.Errorf("profile %q not found (run 'bbkt profile' to list)", name)
		}

		store.ActiveProfile = name
		if err := bitbucket.SaveProfileStore(store); err != nil {
			return fmt.Errorf("saving profile: %w", err)
		}

		fmt.Printf("Active profile set to %q\n", name)
		return nil
	},
}

var profileRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh workspace cache for all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := bitbucket.LoadProfileStore()
		if err != nil {
			return fmt.Errorf("loading profiles: %w", err)
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
			return fmt.Errorf("saving profiles: %w", err)
		}
		fmt.Println("Done.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileRefreshCmd)
}
