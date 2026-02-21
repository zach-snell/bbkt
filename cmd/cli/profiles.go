package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage credential profiles",
	Long:  "List available Bitbucket authentication profiles.",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := bitbucket.LoadProfileStore()
		if err != nil {
			fmt.Println("No profiles found.")
			return
		}

		// Also check what Local Git says so we can indicate auto-detection
		localGitEmail := bitbucket.GetGitUserEmail()

		fmt.Println("Available Profiles:")
		for name, cred := range store.Profiles {
			active := ""
			if name == store.ActiveProfile {
				active = " (default)"
			}
			if localGitEmail != "" && localGitEmail == cred.Email {
				active += " [Git auto-selected]"
			}
			fmt.Printf("  - %s: %s%s\n", name, cred.Email, active)
		}
	},
}

var profileUseCmd = &cobra.Command{
	Use:   "use [name]",
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

func init() {
	RootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileUseCmd)
}
