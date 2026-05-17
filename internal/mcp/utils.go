package mcp

import (
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResolveScope returns the effective workspace and repo for a tool call,
// preferring values the model passed and falling back to BBKT_WORKSPACE /
// BBKT_REPO on the server's environment. BBKT_REPO accepts the same
// "workspace/slug" compound form as the CLI's -R/--repo flag.
//
// This lets a user pin an MCP server to a single workspace in their client
// config (e.g. Claude Desktop) without changing tool schemas. The model can
// always override per-call by passing workspace/repo_slug explicitly.
func ResolveScope(workspace, repo string) (string, string) {
	if workspace == "" {
		workspace = os.Getenv("BBKT_WORKSPACE")
	}
	if repo == "" {
		envRepo := os.Getenv("BBKT_REPO")
		if envRepo != "" {
			if w, s, ok := splitRepoSpec(envRepo); ok {
				if workspace == "" {
					workspace = w
				}
				repo = s
			} else {
				repo = envRepo
			}
		}
	}
	return workspace, repo
}

func splitRepoSpec(spec string) (workspace, slug string, ok bool) {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ToolResultText creates a strictly typed success *mcp.CallToolResult with text content.
func ToolResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// ToolResultError creates a strictly typed error *mcp.CallToolResult.
func ToolResultError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: true,
	}
}
