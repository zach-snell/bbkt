package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageCommitsArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'diff', 'diffstat'" jsonschema_enum:"list,get,diff,diffstat"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Revision  string `json:"revision,omitempty" jsonschema:"Branch name or commit hash to list commits for"`
	Commit    string `json:"commit,omitempty" jsonschema:"Commit hash (required for 'get')"`
	Spec      string `json:"spec,omitempty" jsonschema:"Diff spec: single commit hash or 'hash1..hash2' (required for 'diff', 'diffstat')"`
	Path      string `json:"path,omitempty" jsonschema:"Filter diff/commits to this file path"`
	Include   string `json:"include,omitempty" jsonschema:"Include commits reachable from this ref (for 'list')"`
	Exclude   string `json:"exclude,omitempty" jsonschema:"Exclude commits reachable from this ref (for 'list')"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
}

// ManageCommitsHandler handles the consolidated commit operations.
func ManageCommitsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageCommitsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageCommitsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListCommits(bitbucket.ListCommitsArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Revision:  args.Revision,
				Pagelen:   args.Pagelen,
				Page:      args.Page,
				Include:   args.Include,
				Exclude:   args.Exclude,
				Path:      args.Path,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list commits: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.Commit == "" {
				return ToolResultError("commit is required for 'get' action"), nil, nil
			}
			commit, err := c.GetCommit(bitbucket.GetCommitArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Commit:    args.Commit,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get commit: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(commit, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "diff":
			if args.Spec == "" {
				return ToolResultError("spec is required for 'diff' action"), nil, nil
			}
			raw, err := c.GetDiff(bitbucket.GetDiffArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Spec:      args.Spec,
				Path:      args.Path,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get diff: %v", err)), nil, nil
			}
			return ToolResultText(string(raw)), nil, nil

		case "diffstat":
			if args.Spec == "" {
				return ToolResultError("spec is required for 'diffstat' action"), nil, nil
			}
			result, err := c.GetDiffStat(bitbucket.GetDiffStatArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Spec:      args.Spec,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get diffstat: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
