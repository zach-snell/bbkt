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
// It fails if BITBUCKET_DISABLED_TOOLS names an unknown tool or if the token's
// scopes cannot be introspected (see registerTools).
func New(username, password, token string) (*mcp.Server, error) {
	client := bitbucket.NewClient(username, password, token)
	return newServer(client)
}

// NewFromCredentials creates the MCP server from stored credentials, mapping cached scopes.
// It fails under the same conditions as New.
func NewFromCredentials(creds *bitbucket.Credentials) (*mcp.Server, error) {
	client := bitbucket.NewClientFromCredentials(creds)
	return newServer(client)
}

func newServer(client *bitbucket.Client) (*mcp.Server, error) {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "bbkt",
			Version: version.Version,
		},
		nil,
	)

	if err := registerTools(s, client); err != nil {
		return nil, err
	}
	return s, nil
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

// toolDef describes one MCP tool. toolDefs is the single source of truth for
// the tool set: authenticated and unauthenticated registration and
// BITBUCKET_DISABLED_TOOLS validation all derive from it.
type toolDef struct {
	tool mcp.Tool
	// register adds the tool to s with a handler backed by c.
	register func(s *mcp.Server, c *bitbucket.Client)
	// registerUnauthenticated adds the tool with an auth-required handler.
	registerUnauthenticated func(s *mcp.Server)
}

// defineTool builds a toolDef from a tool's metadata and its handler factory,
// capturing the handler's input type so the input schema is inferred from it.
func defineTool[In any](name, description string, handler func(*bitbucket.Client) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) toolDef {
	tool := mcp.Tool{Name: name, Description: description}
	return toolDef{
		tool: tool,
		register: func(s *mcp.Server, c *bitbucket.Client) {
			t := tool
			mcp.AddTool(s, &t, handler(c))
		},
		registerUnauthenticated: func(s *mcp.Server) {
			addUnauthenticatedTool[In](s, tool)
		},
	}
}

func toolDefs() []toolDef {
	return []toolDef{
		// ─── Workspaces ──────────────────────────────────────────────────
		defineTool("manage_workspaces",
			"Unified tool for getting and listing Bitbucket workspaces",
			ManageWorkspacesHandler),

		// ─── Repositories ────────────────────────────────────────────────
		defineTool("manage_repositories",
			"Unified tool for listing, getting, creating, and deleting repositories",
			ManageRepositoriesHandler),

		// ─── Branches & Tags ─────────────────────────────────────────────
		defineTool("manage_refs",
			"Unified tool for listing, creating, and deleting branches and tags",
			ManageRefsHandler),

		// ─── Commits ─────────────────────────────────────────────────────
		defineTool("manage_commits",
			"Unified tool for listing and getting commits, diffs, and diffstats",
			ManageCommitsHandler),

		// ─── Pull Requests ───────────────────────────────────────────────
		defineTool("manage_pull_requests",
			"Unified tool covering all pull request operations (list, get, create, update, merge, approve, unapprove, decline, diff, diffstat, commits)",
			ManagePullRequestsHandler),

		// ─── PR Comments ─────────────────────────────────────────────────
		defineTool("manage_pr_comments",
			"Unified tool for managing pull request comments (list, create, update, delete, resolve, unresolve)",
			ManagePRCommentsHandler),

		// ─── Source / File Browsing ──────────────────────────────────────
		defineTool("manage_source",
			"Unified tool for source code operations (read, list_directory, get_history, search, write, delete)",
			ManageSourceHandler),

		// ─── Pipelines ───────────────────────────────────────────────────
		defineTool("manage_pipelines",
			"Unified tool for managing Bitbucket Pipelines (list, get, trigger, stop, wait, list-steps, get-step-log)",
			ManagePipelinesHandler),

		// ─── Issues ──────────────────────────────────────────────────────
		defineTool("manage_issues",
			"Unified tool for managing repository issues (list, get, create, update)",
			ManageIssuesHandler),

		// ─── Raw API passthrough (escape hatch) ──────────────────────────
		defineTool("bitbucket_api",
			"Authenticated passthrough to ANY Bitbucket Cloud REST API v2 endpoint not covered by the manage_* tools above (e.g. pipelines-config/OIDC, webhooks, branch restrictions). Prefer a typed manage_* tool when one fits; use this for the long tail instead of calling the API yourself. Reads (GET/HEAD) work by default; write methods are rejected unless BBKT_API_ALLOW_WRITE is set on the server. Supports pagination and returns the raw HTTP status + JSON body.",
			APIRequestHandler),
	}
}

// toolNames returns the name of every tool the server can register, in
// registration order.
func toolNames() []string {
	defs := toolDefs()
	names := make([]string, 0, len(defs))
	for i := range defs {
		names = append(names, defs[i].tool.Name)
	}
	return names
}

// parseDisabledTools parses a comma-separated BITBUCKET_DISABLED_TOOLS value.
// Empty entries (e.g. from a trailing comma) are ignored. Any entry that is not
// a registered tool name is an error: silently ignoring a typo would leave the
// tool exposed while the operator believes it is denied.
func parseDisabledTools(raw string) (map[string]bool, error) {
	valid := toolNames()
	known := make(map[string]bool, len(valid))
	for _, n := range valid {
		known[n] = true
	}

	disabled := make(map[string]bool)
	var unknown []string
	for _, entry := range strings.Split(raw, ",") {
		name := strings.TrimSpace(entry)
		if name == "" {
			continue
		}
		if !known[name] {
			unknown = append(unknown, fmt.Sprintf("%q", name))
			continue
		}
		disabled[name] = true
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("BITBUCKET_DISABLED_TOOLS contains unknown tool name(s): %s (valid names: %s)",
			strings.Join(unknown, ", "), strings.Join(valid, ", "))
	}
	return disabled, nil
}

// skipScopeCheck reports whether BBKT_SKIP_SCOPE_CHECK opts out of failing
// closed when token scope introspection fails.
func skipScopeCheck() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BBKT_SKIP_SCOPE_CHECK"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

const skipScopeCheckHint = "Fix the credentials, or set BBKT_SKIP_SCOPE_CHECK=1 to start anyway with scope filtering disabled (every tool not in BITBUCKET_DISABLED_TOOLS is exposed)."

// NewUnauthenticated creates an MCP server with all tools registered but no working client.
// Every tool call returns an auth-required message. Used for server inspection (e.g., Glama).
func NewUnauthenticated() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "bbkt",
			Version: version.Version,
		},
		nil,
	)
	registerToolsUnauthenticated(s)
	return s
}

const authRequiredMessage = `Authentication required. Configure credentials via:
  1. Run: bbkt auth          (API token — recommended)
  2. Run: bbkt auth --oauth  (OAuth via browser)
  3. Set environment variables (BITBUCKET_ACCESS_TOKEN or BITBUCKET_USERNAME + BITBUCKET_API_TOKEN)
  4. Config file: ~/.config/bbkt/credentials.json`

// addUnauthenticatedTool registers a tool with a handler that always returns an auth-required message.
func addUnauthenticatedTool[In any](s *mcp.Server, tool mcp.Tool) {
	mcp.AddTool(s, &tool, func(_ context.Context, _ *mcp.CallToolRequest, _ In) (*mcp.CallToolResult, any, error) {
		return ToolResultText(authRequiredMessage), nil, nil
	})
}

func registerToolsUnauthenticated(s *mcp.Server) {
	defs := toolDefs()
	for i := range defs {
		defs[i].registerUnauthenticated(s)
	}
}

// registerTools adds every tool that is not disabled and whose required scope
// the token holds. It fails closed: an unknown BITBUCKET_DISABLED_TOOLS entry
// or a failed scope introspection is an error. With BBKT_SKIP_SCOPE_CHECK set,
// an introspection failure only warns and scope filtering is skipped.
func registerTools(s *mcp.Server, c *bitbucket.Client) error {
	disabled, err := parseDisabledTools(os.Getenv("BITBUCKET_DISABLED_TOOLS"))
	if err != nil {
		return err
	}

	tokenScopes, err := c.Scopes()
	if err == nil && len(tokenScopes) == 0 {
		// A successful introspection yields at least one scope; an empty list
		// would make hasRequiredScope allow every tool, so treat it as a failure.
		err = fmt.Errorf("token scope introspection returned no scopes")
	}
	if err != nil {
		if !skipScopeCheck() {
			return fmt.Errorf("failed to introspect token scopes (set BBKT_SKIP_SCOPE_CHECK=1 to bypass): %w\n\n%s", err, skipScopeCheckHint)
		}
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch token scopes for introspection; BBKT_SKIP_SCOPE_CHECK is set, so scope filtering is disabled: %v\n", err)
		tokenScopes = nil // empty scopes make hasRequiredScope allow every tool
	}

	defs := toolDefs()
	for i := range defs {
		d := &defs[i]
		if disabled[d.tool.Name] {
			continue
		}
		if !hasRequiredScope(tokenScopes, getToolRequiredScope(d.tool.Name)) {
			continue // Silently drop the tool if the token lacks the required scope
		}
		d.register(s, c)
	}
	return nil
}
