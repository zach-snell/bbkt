package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
	"github.com/zach-snell/bbkt/internal/version"
)

// New creates and configures the Bitbucket MCP server with all tools registered.
func New(username, password, token string) *mcp.Server {
	client := bitbucket.NewClient(username, password, token)
	return newServer(client)
}

// NewFromCredentials creates the MCP server from stored credentials, mapping cached scopes.
func NewFromCredentials(creds *bitbucket.Credentials) *mcp.Server {
	client := bitbucket.NewClientFromCredentials(creds)
	return newServer(client)
}

func newServer(client *bitbucket.Client) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "bbkt",
			Version: version.Version,
		},
		nil,
	)

	registerTools(s, client)
	return s
}

func getToolRequiredScope(toolName string) []string {
	switch toolName {
	case "manage_workspaces":
		return nil
	case "manage_repositories", "manage_refs", "manage_commits", "manage_source":
		return []string{"repository"}
	case "manage_pull_requests", "manage_pr_comments":
		return []string{"pullrequest"}
	case "manage_pipelines":
		return []string{"pipeline"}
	case "manage_issues":
		return []string{"issue"}
	}
	return nil
}

func hasRequiredScope(tokenScopes, required []string) bool {
	if len(required) == 0 {
		return true
	}
	// Fallback logic for basic app passwords or integrations where we couldn't parse scopes cleanly
	if len(tokenScopes) == 0 {
		return true
	}

	for _, req := range required {
		for _, ts := range tokenScopes {
			// Exact match for standard OAuth formats
			if ts == req {
				return true
			}

			// API Tokens use the pattern `{action}:{resource}:bitbucket`
			// We need to map our internal OAuth-style requirements to these strings.
			switch req {
			case "repository":
				if ts == "repository:write" || ts == "repository:admin" ||
					ts == "read:repository:bitbucket" || ts == "write:repository:bitbucket" || ts == "admin:repository:bitbucket" {
					return true
				}
			case "repository:write":
				if ts == "repository:admin" ||
					ts == "write:repository:bitbucket" || ts == "admin:repository:bitbucket" {
					return true
				}
			case "repository:admin":
				if ts == "admin:repository:bitbucket" {
					return true
				}
			case "pullrequest":
				if ts == "pullrequest:write" ||
					ts == "read:pullrequest:bitbucket" || ts == "write:pullrequest:bitbucket" {
					return true
				}
			case "pullrequest:write":
				if ts == "write:pullrequest:bitbucket" {
					return true
				}
			case "pipeline":
				if ts == "pipeline:write" ||
					ts == "read:pipeline:bitbucket" || ts == "write:pipeline:bitbucket" {
					return true
				}
			case "pipeline:write":
				if ts == "write:pipeline:bitbucket" {
					return true
				}
			case "issue":
				if ts == "issue:write" ||
					ts == "read:issue:bitbucket" || ts == "write:issue:bitbucket" {
					return true
				}
			case "issue:write":
				if ts == "write:issue:bitbucket" {
					return true
				}
			}
		}
	}
	return false
}

// addTool is a helper function to conditionally register a generic tool handler
func addTool[In any](s *mcp.Server, disabled map[string]bool, tokenScopes []string, tool mcp.Tool, handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) {
	if disabled[tool.Name] {
		return
	}
	if !hasRequiredScope(tokenScopes, getToolRequiredScope(tool.Name)) {
		return // Silently drop the tool if the token lacks the required scope
	}
	mcp.AddTool(s, &tool, handler)
}

func registerTools(s *mcp.Server, c *bitbucket.Client) {
	disabledToolsEnv := os.Getenv("BITBUCKET_DISABLED_TOOLS")
	disabled := make(map[string]bool)
	if disabledToolsEnv != "" {
		for _, t := range strings.Split(disabledToolsEnv, ",") {
			disabled[strings.TrimSpace(t)] = true
		}
	}

	tokenScopes, err := c.Scopes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch token scopes for introspection: %v\n", err)
	}

	// ─── Workspaces ──────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_workspaces",
		Description: "Unified tool for getting and listing Bitbucket workspaces",
	}, ManageWorkspacesHandler(c))

	// ─── Repositories ────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_repositories",
		Description: "Unified tool for listing, getting, creating, and deleting repositories",
	}, ManageRepositoriesHandler(c))

	// ─── Branches & Tags ─────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_refs",
		Description: "Unified tool for listing, creating, and deleting branches and tags",
	}, ManageRefsHandler(c))

	// ─── Commits ─────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_commits",
		Description: "Unified tool for listing and getting commits, diffs, and diffstats",
	}, ManageCommitsHandler(c))

	// ─── Pull Requests ───────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_pull_requests",
		Description: "Unified tool covering all pull request operations (list, get, create, update, merge, approve, unapprove, decline, diff, diffstat, commits)",
	}, ManagePullRequestsHandler(c))

	// ─── PR Comments ─────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_pr_comments",
		Description: "Unified tool for managing pull request comments (list, create, update, delete, resolve, unresolve)",
	}, ManagePRCommentsHandler(c))

	// ─── Source / File Browsing ──────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_source",
		Description: "Unified tool for source code operations (read, list_directory, get_history, search, write, delete)",
	}, ManageSourceHandler(c))

	// ─── Pipelines ───────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_pipelines",
		Description: "Unified tool for managing Bitbucket Pipelines (list, get, trigger, stop, list-steps, get-step-log)",
	}, ManagePipelinesHandler(c))

	// ─── Issues ──────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "manage_issues",
		Description: "Unified tool for managing repository issues (list, get, create, update)",
	}, ManageIssuesHandler(c))
}
