package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zach-snell/bitbucket-go-mcp/internal/bitbucket"
)

// New creates and configures the Bitbucket MCP server with all tools registered.
func New(username, password, token string) *server.MCPServer {
	client := bitbucket.NewClient(username, password, token)
	return newServer(client)
}

// NewFromOAuth creates the MCP server from stored OAuth credentials with auto-refresh.
func NewFromOAuth(creds *bitbucket.Credentials) *server.MCPServer {
	client := bitbucket.NewClientFromOAuth(creds)
	return newServer(client)
}

func newServer(client *bitbucket.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"bitbucket-mcp",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithInstructions("Bitbucket Cloud MCP server. Provides tools for interacting with Bitbucket repositories, pull requests, branches, commits, pipelines, and more via the Bitbucket Cloud REST API v2.0."),
	)

	registerTools(s, client)
	return s
}

func registerTools(s *server.MCPServer, c *bitbucket.Client) {
	// ─── Workspaces ──────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_workspaces",
		mcp.WithDescription("List Bitbucket workspaces accessible to the authenticated user"),
		mcp.WithNumber("pagelen", mcp.Description("Number of results per page (default 25, max 100)")),
		mcp.WithNumber("page", mcp.Description("Page number (1-based)")),
	), c.ListWorkspacesHandler)

	s.AddTool(mcp.NewTool("get_workspace",
		mcp.WithDescription("Get details for a Bitbucket workspace"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug or UUID")),
	), c.GetWorkspaceHandler)

	// ─── Repositories ────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_repositories",
		mcp.WithDescription("List repositories in a Bitbucket workspace"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page (default 25)")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithString("query", mcp.Description("Bitbucket query filter (e.g. name~\"myrepo\")")),
		mcp.WithString("role", mcp.Description("Filter by role: owner, admin, contributor, member")),
		mcp.WithString("sort", mcp.Description("Sort field (e.g. -updated_on)")),
	), c.ListRepositoriesHandler)

	s.AddTool(mcp.NewTool("get_repository",
		mcp.WithDescription("Get details for a specific repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
	), c.GetRepositoryHandler)

	s.AddTool(mcp.NewTool("create_repository",
		mcp.WithDescription("Create a new repository in a workspace"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug (URL-friendly name)")),
		mcp.WithString("description", mcp.Description("Repository description")),
		mcp.WithString("language", mcp.Description("Primary programming language")),
		mcp.WithBoolean("is_private", mcp.Description("Whether the repo is private (default true)")),
		mcp.WithString("project_key", mcp.Description("Project key to assign the repo to")),
	), c.CreateRepositoryHandler)

	s.AddTool(mcp.NewTool("delete_repository",
		mcp.WithDescription("Delete a repository (DESTRUCTIVE - cannot be undone)"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
	), c.DeleteRepositoryHandler)

	// ─── Branches ────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_branches",
		mcp.WithDescription("List branches in a repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithString("query", mcp.Description("Filter query")),
		mcp.WithString("sort", mcp.Description("Sort field")),
	), c.ListBranchesHandler)

	s.AddTool(mcp.NewTool("create_branch",
		mcp.WithDescription("Create a new branch from a commit hash"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Branch name")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target commit hash to branch from")),
	), c.CreateBranchHandler)

	s.AddTool(mcp.NewTool("delete_branch",
		mcp.WithDescription("Delete a branch"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Branch name to delete")),
	), c.DeleteBranchHandler)

	// ─── Tags ────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_tags",
		mcp.WithDescription("List tags in a repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
	), c.ListTagsHandler)

	s.AddTool(mcp.NewTool("create_tag",
		mcp.WithDescription("Create a new tag at a specific commit"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Tag name")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target commit hash")),
	), c.CreateTagHandler)

	// ─── Commits ─────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_commits",
		mcp.WithDescription("List commits in a repository, optionally filtered by branch/revision"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("revision", mcp.Description("Branch name or commit hash to list commits for")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithString("include", mcp.Description("Include commits reachable from this ref")),
		mcp.WithString("exclude", mcp.Description("Exclude commits reachable from this ref")),
		mcp.WithString("path", mcp.Description("Filter commits that touch this file path")),
	), c.ListCommitsHandler)

	s.AddTool(mcp.NewTool("get_commit",
		mcp.WithDescription("Get details for a single commit"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("commit", mcp.Required(), mcp.Description("Commit hash")),
	), c.GetCommitHandler)

	s.AddTool(mcp.NewTool("get_diff",
		mcp.WithDescription("Get diff for a commit or between two revisions (e.g. 'hash1..hash2')"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("spec", mcp.Required(), mcp.Description("Diff spec: single commit hash or 'hash1..hash2'")),
		mcp.WithString("path", mcp.Description("Filter diff to this file path")),
	), c.GetDiffHandler)

	s.AddTool(mcp.NewTool("get_diffstat",
		mcp.WithDescription("Get diff statistics (files changed, lines added/removed)"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("spec", mcp.Required(), mcp.Description("Diff spec: single commit hash or 'hash1..hash2'")),
	), c.GetDiffStatHandler)

	// ─── Pull Requests ───────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_pull_requests",
		mcp.WithDescription("List pull requests for a repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("state", mcp.Description("PR state filter: OPEN, MERGED, DECLINED, SUPERSEDED (default OPEN)")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithString("query", mcp.Description("Bitbucket query filter")),
	), c.ListPullRequestsHandler)

	s.AddTool(mcp.NewTool("get_pull_request",
		mcp.WithDescription("Get details for a specific pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.GetPullRequestHandler)

	s.AddTool(mcp.NewTool("create_pull_request",
		mcp.WithDescription("Create a new pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("title", mcp.Required(), mcp.Description("PR title")),
		mcp.WithString("source_branch", mcp.Required(), mcp.Description("Source branch name")),
		mcp.WithString("destination_branch", mcp.Description("Destination branch (defaults to repo main branch)")),
		mcp.WithString("description", mcp.Description("PR description (markdown supported)")),
		mcp.WithBoolean("close_source_branch", mcp.Description("Close source branch after merge")),
		mcp.WithBoolean("draft", mcp.Description("Create as draft PR")),
	), c.CreatePullRequestHandler)

	s.AddTool(mcp.NewTool("update_pull_request",
		mcp.WithDescription("Update a pull request's title or description"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
	), c.UpdatePullRequestHandler)

	s.AddTool(mcp.NewTool("merge_pull_request",
		mcp.WithDescription("Merge a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithString("merge_strategy", mcp.Description("Merge strategy: merge_commit, squash, fast_forward")),
		mcp.WithString("message", mcp.Description("Merge commit message")),
		mcp.WithBoolean("close_source_branch", mcp.Description("Close source branch after merge")),
	), c.MergePullRequestHandler)

	s.AddTool(mcp.NewTool("approve_pull_request",
		mcp.WithDescription("Approve a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.ApprovePullRequestHandler)

	s.AddTool(mcp.NewTool("unapprove_pull_request",
		mcp.WithDescription("Remove approval from a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.UnapprovePullRequestHandler)

	s.AddTool(mcp.NewTool("decline_pull_request",
		mcp.WithDescription("Decline a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.DeclinePullRequestHandler)

	s.AddTool(mcp.NewTool("get_pr_diff",
		mcp.WithDescription("Get the diff for a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.GetPRDiffHandler)

	s.AddTool(mcp.NewTool("get_pr_diffstat",
		mcp.WithDescription("Get diff statistics for a pull request (files changed, lines added/removed)"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.GetPRDiffStatHandler)

	s.AddTool(mcp.NewTool("list_pr_commits",
		mcp.WithDescription("List commits in a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
	), c.ListPRCommitsHandler)

	// ─── PR Comments ─────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_pr_comments",
		mcp.WithDescription("List comments on a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
	), c.ListPRCommentsHandler)

	s.AddTool(mcp.NewTool("create_pr_comment",
		mcp.WithDescription("Add a comment to a pull request. Supports inline comments on specific files/lines and replies to existing comments."),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Comment body (markdown supported)")),
		mcp.WithString("file_path", mcp.Description("File path for inline comment")),
		mcp.WithNumber("line_to", mcp.Description("Line number in new file for inline comment")),
		mcp.WithNumber("line_from", mcp.Description("Line number in old file for inline comment")),
		mcp.WithNumber("parent_id", mcp.Description("Parent comment ID to reply to")),
	), c.CreatePRCommentHandler)

	s.AddTool(mcp.NewTool("update_pr_comment",
		mcp.WithDescription("Update an existing comment on a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithNumber("comment_id", mcp.Required(), mcp.Description("Comment ID")),
		mcp.WithString("content", mcp.Required(), mcp.Description("New comment body")),
	), c.UpdatePRCommentHandler)

	s.AddTool(mcp.NewTool("delete_pr_comment",
		mcp.WithDescription("Delete a comment from a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithNumber("comment_id", mcp.Required(), mcp.Description("Comment ID")),
	), c.DeletePRCommentHandler)

	s.AddTool(mcp.NewTool("resolve_pr_comment",
		mcp.WithDescription("Resolve a comment thread on a pull request"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithNumber("comment_id", mcp.Required(), mcp.Description("Comment ID to resolve")),
	), c.ResolvePRCommentHandler)

	s.AddTool(mcp.NewTool("unresolve_pr_comment",
		mcp.WithDescription("Reopen a resolved comment thread"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pr_id", mcp.Required(), mcp.Description("Pull request ID")),
		mcp.WithNumber("comment_id", mcp.Required(), mcp.Description("Comment ID to reopen")),
	), c.UnresolvePRCommentHandler)

	// ─── Source / File Browsing ──────────────────────────────────────
	s.AddTool(mcp.NewTool("get_file_content",
		mcp.WithDescription("Read a file's content from the repository at a given revision"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path in the repository")),
		mcp.WithString("ref", mcp.Description("Branch name, tag, or commit hash (defaults to HEAD)")),
	), c.GetFileContentHandler)

	s.AddTool(mcp.NewTool("list_directory",
		mcp.WithDescription("List files and directories at a path in the repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("path", mcp.Description("Directory path (empty for root)")),
		mcp.WithString("ref", mcp.Description("Branch name, tag, or commit hash (defaults to HEAD)")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page (default 100)")),
		mcp.WithNumber("max_depth", mcp.Description("Max directory depth to recurse (default 1)")),
	), c.ListDirectoryHandler)

	s.AddTool(mcp.NewTool("get_file_history",
		mcp.WithDescription("Get the commit history for a specific file"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path")),
		mcp.WithString("ref", mcp.Description("Branch/tag/commit to start from (defaults to HEAD)")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
	), c.GetFileHistoryHandler)

	s.AddTool(mcp.NewTool("search_code",
		mcp.WithDescription("Search for code in a repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
	), c.SearchCodeHandler)

	// ─── Pipelines ───────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_pipelines",
		mcp.WithDescription("List pipeline runs for a repository"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithNumber("pagelen", mcp.Description("Results per page")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithString("sort", mcp.Description("Sort field (default -created_on)")),
		mcp.WithString("status", mcp.Description("Filter by status")),
	), c.ListPipelinesHandler)

	s.AddTool(mcp.NewTool("get_pipeline",
		mcp.WithDescription("Get details for a specific pipeline run"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("pipeline_uuid", mcp.Required(), mcp.Description("Pipeline UUID")),
	), c.GetPipelineHandler)

	s.AddTool(mcp.NewTool("trigger_pipeline",
		mcp.WithDescription("Trigger a new pipeline run on a branch"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("ref_name", mcp.Required(), mcp.Description("Branch or tag name to run pipeline on")),
		mcp.WithString("ref_type", mcp.Description("Reference type: branch or tag (default branch)")),
		mcp.WithString("pattern", mcp.Description("Custom pipeline pattern name to trigger")),
	), c.TriggerPipelineHandler)

	s.AddTool(mcp.NewTool("stop_pipeline",
		mcp.WithDescription("Stop a running pipeline"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("pipeline_uuid", mcp.Required(), mcp.Description("Pipeline UUID to stop")),
	), c.StopPipelineHandler)

	s.AddTool(mcp.NewTool("list_pipeline_steps",
		mcp.WithDescription("List steps in a pipeline run"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("pipeline_uuid", mcp.Required(), mcp.Description("Pipeline UUID")),
	), c.ListPipelineStepsHandler)

	s.AddTool(mcp.NewTool("get_pipeline_step_log",
		mcp.WithDescription("Get the log output for a pipeline step"),
		mcp.WithString("workspace", mcp.Required(), mcp.Description("Workspace slug")),
		mcp.WithString("repo_slug", mcp.Required(), mcp.Description("Repository slug")),
		mcp.WithString("pipeline_uuid", mcp.Required(), mcp.Description("Pipeline UUID")),
		mcp.WithString("step_uuid", mcp.Required(), mcp.Description("Step UUID")),
	), c.GetPipelineStepLogHandler)
}
