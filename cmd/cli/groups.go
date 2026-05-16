package cli

import "github.com/spf13/cobra"

// Group IDs for cobra command grouping. Set on each command via cmd.GroupID
// and registered on RootCmd in registerGroups() below.
const (
	groupAuth = "auth"
	groupData = "data"
	groupMCP  = "mcp"
)

// registerGroups adds the top-level command groups so `bbkt --help` renders
// commands in a sensible order (auth/profiles first, resources next, MCP
// server last) instead of alphabetically. Built-in cobra commands (help,
// completion) stay ungrouped at the bottom.
func registerGroups() {
	RootCmd.AddGroup(
		&cobra.Group{ID: groupAuth, Title: "Authentication & Profiles:"},
		&cobra.Group{ID: groupData, Title: "Bitbucket Resources:"},
		&cobra.Group{ID: groupMCP, Title: "MCP Server:"},
	)
}

// addPaginationFlags wires --page and --pagelen onto a list-style command.
// Bitbucket's REST envelope returns at most 100 per page; we don't enforce
// here because the server already does.
func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 0, "Page number (1-based; 0 = first page)")
	cmd.Flags().Int("pagelen", 0, "Results per page (0 = server default, max 100)")
}

// paginationArgs reads --page/--pagelen off a command's flags. Returns
// zeros when the flags aren't registered (safe to call from any RunE).
func paginationArgs(cmd *cobra.Command) (page, pagelen int) {
	page, _ = cmd.Flags().GetInt("page")
	pagelen, _ = cmd.Flags().GetInt("pagelen")
	return
}
