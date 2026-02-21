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

It allows you to manage workspaces, repositories, pull requests,
pipelines, and more directly from your terminal, or expose these
capabilities to your AI agents via the MCP protocol.

Try running 'bbkt auth' to get started!`,
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
	// Configure global flags here
	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bbkt.yaml)")
}
