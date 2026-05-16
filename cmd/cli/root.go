package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/version"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "bbkt",
	Version: version.Version,
	Short:   "A unified CLI and MCP server for Bitbucket Cloud",
	Long: `bbkt is a complete command-line interface and Model Context Protocol
server for Bitbucket Cloud.

Manage workspaces, repositories, pull requests, pipelines, and source
code from your terminal, or expose the same capabilities to AI agents
via 'bbkt mcp'.

Most commands auto-detect the current workspace/repo from your git
config when run inside a clone, so 'bbkt prs list' Just Works.

Credentials are stored at ~/.config/bbkt/credentials.json. Use
--profile (or BBKT_PROFILE) to switch between named profiles
(e.g. personal vs work).`,
	Example: `  # First-time setup
  bbkt auth                        # save credentials to default profile
  bbkt status                      # confirm you're authenticated

  # Daily use (inside a Bitbucket repo)
  bbkt prs list                    # workspace/repo inferred from git
  bbkt pipelines trigger -r main

  # Multi-account
  bbkt auth --profile work
  bbkt --profile work prs list     # one-shot profile override

  # MCP server (for Claude Desktop / Cursor / etc.)
  bbkt mcp`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if profile, _ := cmd.Flags().GetString("profile"); profile != "" {
			os.Setenv("BBKT_PROFILE", profile)
		}
	},
	// Errors from RunE shouldn't dump the usage wall — cobra's default
	// is "print error + usage" which is overwhelming for a runtime API
	// failure. SilenceErrors=true keeps cobra from printing the error
	// itself, since Execute() already prints it. SilenceUsage=true keeps
	// the help/usage wall from following.
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().Bool("json", false, "Output raw JSON instead of formatted tables")
	RootCmd.PersistentFlags().StringP("profile", "p", "", "Credential profile to use (overrides active profile / BBKT_PROFILE)")
	registerGroups()
}
