package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
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
			Name:    "bitbucket-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	registerTools(s, client)
	return s
}

func getToolRequiredScope(toolName string) []string {
	switch toolName {
	case "list_workspaces", "get_workspace":
		return nil // Basic read access implies these are readable
	case "list_repositories", "get_repository", "list_branches", "list_tags", "list_commits", "get_commit", "get_diff", "get_diffstat", "get_file_content", "list_directory", "get_file_history", "search_code":
		return []string{"repository"}
	case "create_repository", "delete_repository", "create_branch", "delete_branch", "create_tag", "write_file", "delete_file":
		if toolName == "delete_repository" {
			return []string{"repository:delete"}
		}
		return []string{"repository:write", "repository:admin"}
	case "list_pull_requests", "get_pull_request", "get_pr_diff", "get_pr_diffstat", "list_pr_commits", "list_pr_comments":
		return []string{"pullrequest"}
	case "create_pull_request", "update_pull_request", "merge_pull_request", "approve_pull_request", "unapprove_pull_request", "decline_pull_request", "create_pr_comment", "update_pr_comment", "delete_pr_comment", "resolve_pr_comment", "unresolve_pr_comment":
		return []string{"pullrequest:write"}
	case "list_pipelines", "get_pipeline", "list_pipeline_steps", "get_pipeline_step_log":
		return []string{"pipeline"}
	case "trigger_pipeline", "stop_pipeline":
		return []string{"pipeline:write"}
	case "list_issues", "get_issue":
		return []string{"issue"}
	case "create_issue", "update_issue":
		return []string{"issue:write"}
	}
	return nil
}

func hasRequiredScope(tokenScopes []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	// Fallback logic for basic app passwords or integrations where we couldn't parse scopes cleanly
	if len(tokenScopes) == 0 {
		return true
	}

	for _, req := range required {
		for _, ts := range tokenScopes {
			// Exact match
			if ts == req {
				return true
			}
			// Administrative scopes imply read/write scopes
			if req == "repository" && (ts == "repository:write" || ts == "repository:admin") {
				return true
			}
			if req == "repository:write" && ts == "repository:admin" {
				return true
			}
			if req == "pullrequest" && ts == "pullrequest:write" {
				return true
			}
			if req == "issue" && ts == "issue:write" {
				return true
			}
			if req == "pipeline" && ts == "pipeline:write" {
				return true
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
		Name:        "list_workspaces",
		Description: "List Bitbucket workspaces accessible to the authenticated user",
	}, ListWorkspacesHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_workspace",
		Description: "Get details for a Bitbucket workspace",
	}, GetWorkspaceHandler(c))

	// ─── Repositories ────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_repositories",
		Description: "List repositories in a Bitbucket workspace",
	}, ListRepositoriesHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_repository",
		Description: "Get details for a specific repository",
	}, GetRepositoryHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_repository",
		Description: "Create a new repository in a workspace",
	}, CreateRepositoryHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "delete_repository",
		Description: "Delete a repository (DESTRUCTIVE - cannot be undone)",
	}, DeleteRepositoryHandler(c))

	// ─── Branches ────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_branches",
		Description: "List branches in a repository",
	}, ListBranchesHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_branch",
		Description: "Create a new branch from a commit hash",
	}, CreateBranchHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "delete_branch",
		Description: "Delete a branch",
	}, DeleteBranchHandler(c))

	// ─── Tags ────────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_tags",
		Description: "List tags in a repository",
	}, ListTagsHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_tag",
		Description: "Create a new tag at a specific commit",
	}, CreateTagHandler(c))

	// ─── Commits ─────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_commits",
		Description: "List commits in a repository, optionally filtered by branch/revision",
	}, ListCommitsHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_commit",
		Description: "Get details for a single commit",
	}, GetCommitHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_diff",
		Description: "Get diff for a commit or between two revisions (e.g. 'hash1..hash2')",
	}, GetDiffHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_diffstat",
		Description: "Get diff statistics (files changed, lines added/removed)",
	}, GetDiffStatHandler(c))

	// ─── Pull Requests ───────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_pull_requests",
		Description: "List pull requests for a repository",
	}, ListPullRequestsHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_pull_request",
		Description: "Get details for a specific pull request",
	}, GetPullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_pull_request",
		Description: "Create a new pull request",
	}, CreatePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "update_pull_request",
		Description: "Update a pull request's title or description",
	}, UpdatePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "merge_pull_request",
		Description: "Merge a pull request",
	}, MergePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "approve_pull_request",
		Description: "Approve a pull request",
	}, ApprovePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "unapprove_pull_request",
		Description: "Remove approval from a pull request",
	}, UnapprovePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "decline_pull_request",
		Description: "Decline a pull request",
	}, DeclinePullRequestHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_pr_diff",
		Description: "Get the diff for a pull request",
	}, GetPRDiffHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_pr_diffstat",
		Description: "Get diff statistics for a pull request (files changed, lines added/removed)",
	}, GetPRDiffStatHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_pr_commits",
		Description: "List commits in a pull request",
	}, ListPRCommitsHandler(c))

	// ─── PR Comments ─────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_pr_comments",
		Description: "List comments on a pull request",
	}, ListPRCommentsHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_pr_comment",
		Description: "Add a comment to a pull request. Supports inline comments on specific files/lines and replies to existing comments.",
	}, CreatePRCommentHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "update_pr_comment",
		Description: "Update an existing comment on a pull request",
	}, UpdatePRCommentHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "delete_pr_comment",
		Description: "Delete a comment from a pull request",
	}, DeletePRCommentHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "resolve_pr_comment",
		Description: "Resolve a comment thread on a pull request",
	}, ResolvePRCommentHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "unresolve_pr_comment",
		Description: "Reopen a resolved comment thread",
	}, UnresolvePRCommentHandler(c))

	// ─── Source / File Browsing ──────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_file_content",
		Description: "Read a file's content from the repository at a given revision",
	}, GetFileContentHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_directory",
		Description: "List files and directories at a path in the repository",
	}, ListDirectoryHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_file_history",
		Description: "Get the commit history for a specific file",
	}, GetFileHistoryHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "search_code",
		Description: "Search for code in a repository",
	}, SearchCodeHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "write_file",
		Description: "Write or update a file in the repository",
	}, WriteFileHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file from the repository",
	}, DeleteFileHandler(c))

	// ─── Pipelines ───────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_pipelines",
		Description: "List pipeline runs for a repository",
	}, ListPipelinesHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_pipeline",
		Description: "Get details for a specific pipeline run",
	}, GetPipelineHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "trigger_pipeline",
		Description: "Trigger a new pipeline run on a branch",
	}, TriggerPipelineHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "stop_pipeline",
		Description: "Stop a running pipeline",
	}, StopPipelineHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_pipeline_steps",
		Description: "List steps in a pipeline run",
	}, ListPipelineStepsHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_pipeline_step_log",
		Description: "Get the log output for a pipeline step",
	}, GetPipelineStepLogHandler(c))

	// ─── Issues ──────────────────────────────────────────────────────
	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "list_issues",
		Description: "List issues in a repository",
	}, ListIssuesHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "get_issue",
		Description: "Get details for a specific issue",
	}, GetIssueHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new issue",
	}, CreateIssueHandler(c))

	addTool(s, disabled, tokenScopes, mcp.Tool{
		Name:        "update_issue",
		Description: "Update an existing issue",
	}, UpdateIssueHandler(c))
}
