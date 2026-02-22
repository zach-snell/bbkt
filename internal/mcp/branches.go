package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageRefsArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list-branches', 'create-branch', 'delete-branch', 'list-tags', 'create-tag'" jsonschema_enum:"list-branches,create-branch,delete-branch,list-tags,create-tag"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Name      string `json:"name,omitempty" jsonschema:"Branch or tag name (required for create/delete)"`
	Target    string `json:"target,omitempty" jsonschema:"Target commit hash (required for create-tag)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (for list)"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (for list)"`
	Query     string `json:"query,omitempty" jsonschema:"Filter query (for list-branches)"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field (for list-branches)"`
}

// ManageRefsHandler handles the consolidated branch and tag operations.
func ManageRefsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageRefsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageRefsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list-branches":
			result, err := c.ListBranches(bitbucket.ListBranchesArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Pagelen:   args.Pagelen,
				Page:      args.Page,
				Query:     args.Query,
				Sort:      args.Sort,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list branches: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create-branch":
			if args.Name == "" || args.Target == "" {
				return ToolResultError("name and target are required for 'create-branch' action"), nil, nil
			}
			branch, err := c.CreateBranch(bitbucket.CreateBranchArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Name:      args.Name,
				Target:    args.Target,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create branch: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(branch, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "delete-branch":
			if args.Name == "" {
				return ToolResultError("name is required for 'delete-branch' action"), nil, nil
			}
			err := c.DeleteBranch(bitbucket.DeleteBranchArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Name:      args.Name,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete branch: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Branch '%s' deleted successfully", args.Name)), nil, nil

		case "list-tags":
			result, err := c.ListTags(bitbucket.ListTagsArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Pagelen:   args.Pagelen,
				Page:      args.Page,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list tags: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create-tag":
			if args.Name == "" || args.Target == "" {
				return ToolResultError("name and target are required for 'create-tag' action"), nil, nil
			}
			tag, err := c.CreateTag(bitbucket.CreateTagArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Name:      args.Name,
				Target:    args.Target,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create tag: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(tag, "", "  ")
			return ToolResultText(string(data)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
